package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"autowifi/internal/agent"
	"autowifi/internal/autostart"
	"autowifi/internal/config"
	"autowifi/internal/creds"
	"autowifi/internal/engine"
	"autowifi/internal/notify"
	"autowifi/internal/portal"
	"autowifi/internal/prompt"
	"autowifi/internal/state"
	"autowifi/internal/tray"
	"autowifi/internal/update"
)

// env is everything realMain touches outside its own memory, so tests can replace it.
type env struct {
	ctx            context.Context
	stdout, stderr io.Writer
	configDir      string // default home of config.json, state.json, log, lock, kick files
	home           string
	exe            string
	store          creds.Store
	notifier       notify.Notifier
	interactive    bool                              // stdin is a terminal
	askTerminal    func() (creds.Credentials, error) // used when interactive
	askDialog      func() (creds.Credentials, error) // used otherwise
	newClient      func(config.PortalConfig) (*portal.Client, error)
	autostartOn    func(home, exe string) error
	autostartOff   func(home string) error
	autostartIsOn  func(home string) (bool, error)
	after          func(time.Duration) <-chan time.Time // nil = real time
	pollInterval   time.Duration                        // 0 = agent default
	kickInterval   time.Duration                        // how often run looks for kick files
	trayAvailable  bool
	updateInterval time.Duration
}

// osEnv is the real environment.
func osEnv(ctx context.Context) (*env, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	exe, err := os.Executable()
	if err == nil {
		if r, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = r
		}
	}
	e := &env{
		ctx: ctx, stdout: os.Stdout, stderr: os.Stderr, configDir: dir, home: home, exe: exe,
		notifier:    notify.NewSystem(),
		interactive: prompt.IsTerminal(),
		askTerminal: prompt.NewTerminal().Ask,
		askDialog:   prompt.Dialog,
		newClient:   portal.NewClient,
		autostartOn: autostart.Install, autostartOff: autostart.Uninstall, autostartIsOn: autostart.Installed,
		kickInterval: 2 * time.Second, trayAvailable: true, updateInterval: 24 * time.Hour,
	}
	if runtime.GOOS == "linux" {
		// The Secret Service probe (and the fallback warning) only happens when a command
		// actually touches credentials, not for version/help/capture/autostart.
		fb := filepath.Join(dir, "credentials")
		e.store = &lazyStore{make: func() creds.Store {
			return creds.NewDefault(creds.KeyringProbe, fb, func(msg string) {
				fmt.Fprintln(os.Stderr, "autowifi:", msg)
				_ = e.notifier.Notify("AutoWiFi BITS", msg)
			})
		}}
		// fyne.io/systray needs a D-Bus session and a display; without them run headless.
		e.trayAvailable = os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" &&
			(os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "")
	} else {
		e.store = creds.NewKeyring()
	}
	return e, nil
}

// lazyStore builds the real store on first use.
type lazyStore struct {
	once sync.Once
	make func() creds.Store
	s    creds.Store
}

func (l *lazyStore) get() creds.Store                { l.once.Do(func() { l.s = l.make() }); return l.s }
func (l *lazyStore) Get() (creds.Credentials, error) { return l.get().Get() }
func (l *lazyStore) Set(c creds.Credentials) error   { return l.get().Set(c) }
func (l *lazyStore) Delete() error                   { return l.get().Delete() }

const usage = `usage: autowifi [--config <path>] <command>

  run [--no-tray]     background agent (default with no command)
  setup               store your campus ID and password in the OS secure store
  login-now           check the portal and log in now
  logout              end the portal session
  capture             show the portal form fields and certificate (never values)
  status              show what the agent knows
  autostart on|off    start at login, or not
  forget              delete the stored ID and password
  version             print the version
`

// paths holds the files the commands share.
type paths struct {
	config, state, log, lock string
	dir                      string
}

func (p paths) kick(name string) string { return filepath.Join(p.dir, "kick-"+name) }

// commandOf returns the subcommand realMain will run for args ("run" when none is given), or ""
// when the flags do not parse. main uses it to decide on signal handling and the console.
func commandOf(args []string) string {
	fs := flag.NewFlagSet("autowifi", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.String("config", "", "")
	if fs.Parse(args) != nil {
		return ""
	}
	if fs.NArg() == 0 {
		return "run"
	}
	return fs.Arg(0)
}

func realMain(args []string, e *env) int {
	fs := flag.NewFlagSet("autowifi", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfgPath := fs.String("config", "", "config file (default "+filepath.Join(e.configDir, "config.json")+")")
	fs.Usage = func() {} // -h is handled below; errors print usage themselves
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(e.stdout, usage)
			return 0
		}
		fmt.Fprintf(e.stderr, "autowifi: %v\n\n%s", err, usage)
		return 2
	}
	p := paths{config: *cfgPath, dir: e.configDir}
	if p.config == "" {
		p.config = filepath.Join(e.configDir, "config.json")
	} else {
		p.dir = filepath.Dir(p.config)
	}
	p.state = filepath.Join(p.dir, "state.json")
	p.log = filepath.Join(p.dir, "autowifi.log")
	p.lock = filepath.Join(p.dir, "agent.lock")

	rest := fs.Args()
	cmd := "run"
	if len(rest) > 0 {
		cmd, rest = rest[0], rest[1:]
	}
	switch cmd {
	case "run":
		return cmdRun(e, p, rest)
	case "setup":
		return cmdSetup(e, p)
	case "login-now":
		return cmdLoginNow(e, p)
	case "logout":
		return cmdLogout(e, p)
	case "capture":
		return cmdCapture(e, p)
	case "status":
		return cmdStatus(e, p)
	case "autostart":
		return cmdAutostart(e, rest)
	case "forget":
		if err := e.store.Delete(); err != nil {
			return fail(e, "could not delete the stored ID and password: %v", err)
		}
		fmt.Fprintln(e.stdout, "Stored ID and password deleted.")
		return 0
	case "version":
		fmt.Fprintln(e.stdout, version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(e.stdout, usage)
		return 0
	}
	fmt.Fprintf(e.stderr, "autowifi: unknown command %q\n\n%s", cmd, usage)
	return 2
}

func fail(e *env, format string, a ...any) int {
	fmt.Fprintf(e.stderr, "autowifi: "+format+"\n", a...)
	return 1
}

// loadEngine builds an Engine from the config and the saved state.
func loadEngine(e *env, p paths, logf func(string, ...any)) (*engine.Engine, *portal.Client, error) {
	cfg, err := config.Load(p.config)
	if err != nil {
		return nil, nil, fmt.Errorf("config %s: %w", p.config, err)
	}
	client, err := e.newClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	st, err := state.Load(p.state)
	if err != nil {
		return nil, nil, err
	}
	en := &engine.Engine{
		Portal: client, Creds: e.store, Notifier: e.notifier, State: st, Now: time.Now,
		Save: func(s state.SessionState) error { return state.Save(p.state, s) },
		Log:  logf,
	}
	return en, client, nil
}

func askCredentials(e *env) (creds.Credentials, error) {
	if e.interactive {
		return e.askTerminal()
	}
	return e.askDialog()
}

// touch creates an empty marker file (a request to the running agent).
func touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0o600)
}

// takeKick removes the marker and reports whether it existed.
func takeKick(path string) bool { return os.Remove(path) == nil }

func cmdSetup(e *env, p paths) int {
	c, err := askCredentials(e)
	if errors.Is(err, prompt.ErrCancelled) {
		fmt.Fprintln(e.stderr, "Cancelled; nothing saved.")
		return 1
	}
	if err != nil {
		return fail(e, "%v", err)
	}
	if err := e.store.Set(c); err != nil {
		return fail(e, "could not store the ID and password: %v", err)
	}
	_ = touch(p.kick("creds")) // a running agent un-pauses and logs in with the new ones
	fmt.Fprintln(e.stdout, "Saved in the OS secure store. AutoWiFi BITS will use them on the next check.")
	return 0
}

// oneShot runs f with an Engine when no agent is running; otherwise it leaves a kick file for
// the agent (so two processes never write state.json at once).
func oneShot(e *env, p paths, kick string, f func(*engine.Engine) int) int {
	release, ok, err := acquireLock(p.lock)
	if err != nil {
		return fail(e, "%v", err)
	}
	if !ok {
		if err := touch(p.kick(kick)); err != nil {
			return fail(e, "%v", err)
		}
		fmt.Fprintln(e.stdout, "Asked the running AutoWiFi BITS agent to do it.")
		return 0
	}
	defer release()
	en, _, err := loadEngine(e, p, func(string, ...any) {})
	if err != nil {
		return fail(e, "%v", err)
	}
	return f(en)
}

func cmdLoginNow(e *env, p paths) int {
	return oneShot(e, p, "manual", func(en *engine.Engine) int {
		s := en.Run(e.ctx, engine.TriggerManual)
		fmt.Fprintf(e.stdout, "%s (%s)\n", tray.StatusText(s.Outcome), s.Outcome)
		switch s.Outcome {
		case portal.Rejected, portal.FormChanged, portal.PortalError, portal.NoCredentials:
			return 1
		}
		return 0
	})
}

func cmdLogout(e *env, p paths) int {
	return oneShot(e, p, "logout", func(en *engine.Engine) int {
		was := en.State.LoggedIn()
		en.Logout(e.ctx)
		if was {
			fmt.Fprintln(e.stdout, "Logged out.")
		} else {
			fmt.Fprintln(e.stdout, "No portal session to log out of.")
		}
		return 0
	})
}

func cmdCapture(e *env, p paths) int {
	cfg, err := config.Load(p.config)
	if err != nil {
		return fail(e, "config %s: %v", p.config, err)
	}
	c, err := e.newClient(cfg)
	if err != nil {
		return fail(e, "%v", err)
	}
	r, err := c.Capture(e.ctx)
	if err != nil {
		return fail(e, "probe failed: %v", err)
	}
	fmt.Fprint(e.stdout, portal.FormatCapture(r))
	return 0
}

func cmdStatus(e *env, p paths) int {
	st, err := state.Load(p.state)
	if err != nil {
		return fail(e, "%v", err)
	}
	yes := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	_, credErr := e.store.Get()
	stored := yes(credErr == nil)
	if credErr != nil && !errors.Is(credErr, creds.ErrNotFound) {
		stored = "unknown (" + credErr.Error() + ")"
	}
	running := false
	if release, ok, err := acquireLock(p.lock); err == nil {
		if ok {
			release()
		} else {
			running = true
		}
	}
	auto, _ := e.autostartIsOn(e.home)
	last := "none yet"
	if b, err := os.ReadFile(filepath.Join(p.dir, "last-outcome")); err == nil {
		last = strings.TrimSpace(string(b))
	}
	fmt.Fprintf(e.stdout, "Agent running: %s\nStart at login: %s\nID and password stored: %s\nCredentials rejected: %s\nPortal session: %s\nLast check: %s\nConfig: %s\nVersion: %s\n",
		yes(running), yes(auto), stored, yes(st.CredentialsRejected), yes(st.LoggedIn()), last, p.config, version)
	return 0
}

func cmdAutostart(e *env, rest []string) int {
	if len(rest) != 1 || (rest[0] != "on" && rest[0] != "off") {
		fmt.Fprintln(e.stderr, "usage: autowifi autostart on|off")
		return 2
	}
	if rest[0] == "on" {
		if err := e.autostartOn(e.home, e.exe); err != nil {
			return fail(e, "could not enable start at login: %v", err)
		}
		fmt.Fprintln(e.stdout, "AutoWiFi BITS will start when you log in.")
		return 0
	}
	if err := e.autostartOff(e.home); err != nil {
		return fail(e, "could not disable start at login: %v", err)
	}
	fmt.Fprintln(e.stdout, "Start at login is off.")
	return 0
}

func cmdRun(e *env, p paths, rest []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	noTray := fs.Bool("no-tray", false, "run without a tray icon")
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	useTray := e.trayAvailable && !*noTray

	release, ok, err := acquireLock(p.lock)
	if err != nil {
		return fail(e, "%v", err)
	}
	if !ok {
		fmt.Fprintln(e.stdout, "AutoWiFi BITS is already running.")
		return 0
	}
	defer release()

	logger, closeLog, err := openLog(p.log)
	if err != nil {
		return fail(e, "log: %v", err)
	}
	defer closeLog()
	logf := func(f string, a ...any) { logger.Printf(f, a...) }

	en, _, err := loadEngine(e, p, logf)
	if err != nil {
		logf("start failed: %v", err)
		return fail(e, "%v", err)
	}
	logf("agent %s starting", version)

	// First run without credentials: ask once (terminal or dialog) and turn on start at login.
	firstRun := false
	if _, err := os.Stat(p.state); errors.Is(err, os.ErrNotExist) {
		firstRun = true
	}
	if !firstRun {
		// nothing to ask
	} else if _, err := e.store.Get(); errors.Is(err, creds.ErrNotFound) {
		if c, err := askCredentials(e); err != nil {
			if !errors.Is(err, prompt.ErrCancelled) {
				logf("first-run credential prompt: %v", err)
			}
		} else {
			if err := e.store.Set(c); err != nil {
				logf("store credentials: %v", err)
			} else if e.exe != "" {
				if err := e.autostartOn(e.home, e.exe); err != nil {
					logf("autostart: %v", err)
				} else {
					logf("start at login enabled on first run")
				}
			}
		}
	}
	_ = state.Save(p.state, en.State) // marks the first run as done

	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()

	var (
		trMu       sync.Mutex
		tr         *tray.Tray
		lastStatus portal.Outcome
	)
	setStatus := func(o portal.Outcome) {
		trMu.Lock()
		defer trMu.Unlock()
		lastStatus = o
		if tr != nil {
			tr.SetStatus(o)
		}
	}
	a := &agent.Agent{
		Engine: en, After: e.after, PollInterval: e.pollInterval,
		OnStep: func(t engine.Trigger, s engine.Step) {
			logf("next check in %s", s.Next.Round(time.Second)) // the engine already logged the outcome
			_ = os.WriteFile(filepath.Join(p.dir, "last-outcome"),
				[]byte(fmt.Sprintf("%s at %s\n", s.Outcome, time.Now().Format(time.RFC3339))), 0o600)
			setStatus(s.Outcome)
		},
	}
	go watchKicks(ctx, p, a, e.kickInterval)
	go checkUpdates(ctx, logf, e.updateInterval, func(tag, url string) {
		trMu.Lock()
		defer trMu.Unlock()
		if tr != nil {
			tr.SetUpdate(tag, url)
		}
	})

	if !useTray {
		_ = a.Run(ctx)
		logf("agent stopped")
		return 0
	}

	done := make(chan struct{})
	go func() {
		_ = a.Run(ctx)
		close(done)
	}()
	go func() { <-ctx.Done(); tray.Quit() }()
	tray.Run(tray.Handlers{
		LoginNow: func() { a.Kick(engine.TriggerManual) },
		Logout:   a.Logout,
		SetCredentials: func() {
			go func() {
				c, err := e.askDialog()
				if errors.Is(err, prompt.ErrCancelled) {
					return
				}
				if err != nil {
					logf("credential dialog: %v", err)
					_ = e.notifier.Notify("AutoWiFi BITS", "Could not open the ID and password dialog. Run 'autowifi setup' in a terminal.")
					return
				}
				if err := e.store.Set(c); err != nil {
					logf("store credentials: %v", err)
					_ = e.notifier.Notify("AutoWiFi BITS", "Your ID and password could not be saved in the OS secure store.")
					return
				}
				a.Kick(engine.TriggerCredsUpdated)
			}()
		},
		Quit: cancel,
		Autostart: func() bool {
			on, _ := e.autostartIsOn(e.home)
			return on
		},
		SetAutostart: func(on bool) error {
			var err error
			if on {
				err = e.autostartOn(e.home, e.exe)
			} else {
				err = e.autostartOff(e.home)
			}
			if err != nil {
				logf("start at login: %v", err)
				_ = e.notifier.Notify("AutoWiFi BITS", "Could not change start at login. Try 'autowifi autostart on' in a terminal.")
			}
			return err
		},
	}, func(t *tray.Tray) {
		trMu.Lock()
		defer trMu.Unlock()
		tr = t
		if lastStatus != "" { // the first check may finish before the menu exists
			t.SetStatus(lastStatus)
		}
	})
	trMu.Lock()
	tr = nil // the native menu is gone; late callbacks must not touch it
	trMu.Unlock()
	cancel()
	<-done
	logf("agent stopped")
	return 0
}

// watchKicks turns marker files left by other autowifi processes into agent triggers.
func watchKicks(ctx context.Context, p paths, a *agent.Agent, every time.Duration) {
	if every <= 0 {
		every = 2 * time.Second
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if takeKick(p.kick("creds")) {
				a.Kick(engine.TriggerCredsUpdated)
			}
			if takeKick(p.kick("manual")) {
				a.Kick(engine.TriggerManual)
			}
			if takeKick(p.kick("logout")) {
				a.Logout()
			}
		}
	}
}

// checkUpdates asks GitHub once a minute after start and then every interval. It only informs.
func checkUpdates(ctx context.Context, logf func(string, ...any), interval time.Duration, found func(tag, url string)) {
	if repo == "" || version == "dev" || interval <= 0 {
		return
	}
	api := "https://api.github.com/repos/" + repo + "/releases/latest"
	client := &http.Client{Timeout: 20 * time.Second}
	wait := time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = interval
		rel, err := update.Latest(ctx, client, api)
		if err != nil {
			logf("update check: %v", err)
			continue
		}
		if update.Newer(version, rel.Tag) {
			logf("update available: %s", rel.Tag)
			found(rel.Tag, rel.URL)
		}
	}
}

// openLog opens the agent log (mode 0600), rotated once at 1 MiB.
func openLog(path string) (*log.Logger, func(), error) {
	w, err := newRotWriter(path, 1<<20)
	if err != nil {
		return nil, nil, err
	}
	return log.New(w, "", log.LstdFlags), func() { w.Close() }, nil
}
