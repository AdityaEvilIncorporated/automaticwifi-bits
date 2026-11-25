#!/bin/bash
# Local database and services helper. Claude, workers, the gate and prep all use it.
# It only ever talks to the Postgres in this project's docker compose (localhost).
#
#   bash scripts/db.sh ensure          start Docker (if needed) and every service in COMPOSE_PROFILES, wait until healthy
#   bash scripts/db.sh wait            wait until Postgres accepts connections
#   bash scripts/db.sh create <db>     create a database if it does not exist
#   bash scripts/db.sh drop <db>       drop a database (refuses the main one without --force)
#   bash scripts/db.sh url <db>        print the connection URL for <db>
#   bash scripts/db.sh migrate [db]    apply migrations (scripts/migrate.sh, else auto-detect the tool)
#   bash scripts/db.sh seed [db]       create admin/test users and sample data (scripts/seed.sh)
#   bash scripts/db.sh fresh <db>      drop + create + migrate (the gate uses this per task)
#   bash scripts/db.sh reset [db]      drop + create + migrate + seed
#   bash scripts/db.sh sql [db] "<sql>"   run SQL (also: psql [db] for a prompt)
#   bash scripts/db.sh status          services and databases
set -e
ROOT=$(cd "$(dirname "$0")/.." && pwd); cd "$ROOT"
if [ -f .env ]; then
  # Load .env without overriding variables already set (the gate sets a per-task DATABASE_URL).
  while IFS='=' read -r k v; do
    [[ $k =~ ^[A-Z0-9_]+$ ]] || continue
    [ -z "${!k+x}" ] && export "$k=$v"
  done < .env
fi
export PGOPTIONS="-c client_min_messages=warning"
U=${POSTGRES_USER:-app}; PW=${POSTGRES_PASSWORD:-}; PORT=${DB_PORT:-5432}; MAIN=${POSTGRES_DB:-app}
url() { echo "postgresql://$U:$PW@localhost:$PORT/$1"; }
valid() { [[ $1 =~ ^[a-z][a-z0-9_]{0,62}$ ]] || { echo "bad database name: $1 (lowercase, digits, _)" >&2; exit 2; }; }

have_compose_db() { [ -f docker-compose.yml ] && command -v docker >/dev/null && docker compose ps --services --status running 2>/dev/null | grep -qx db; }
psql_() {  # psql_ <db> [psql args]
  local db=$1; shift
  if [ -z "$BS_DB_HOST_PSQL" ] && have_compose_db; then
    docker compose exec -T -e PGOPTIONS db psql -X -v ON_ERROR_STOP=1 -q -U "$U" -d "$db" "$@"
  else
    PGPASSWORD=$PW psql -X -v ON_ERROR_STOP=1 -q -h localhost -p "$PORT" -U "$U" -d "$db" "$@"
  fi
}

start_docker() {
  docker info >/dev/null 2>&1 && return 0
  [ "$(uname)" = Darwin ] && open -ga Docker 2>/dev/null
  for _ in $(seq 1 "${BS_DOCKER_WAIT:-90}"); do docker info >/dev/null 2>&1 && return 0; sleep 2; done
  echo "Docker is not running and did not start within 3 minutes" >&2; return 1
}

wait_db() {
  for _ in $(seq 1 60); do psql_ postgres -tAc 'select 1' >/dev/null 2>&1 && return 0; sleep 1; done
  echo "Postgres did not answer within 60 seconds (bash scripts/db.sh status)" >&2; return 1
}

migrate() {
  local db=${1:-$MAIN}; valid "$db"
  export DATABASE_URL; DATABASE_URL=$(url "$db")
  if [ -s scripts/migrate.sh ]; then bash scripts/migrate.sh; return; fi
  if [ -f alembic.ini ]; then
    if command -v uv >/dev/null && [ -f pyproject.toml ]; then uv run alembic upgrade head; else alembic upgrade head; fi
  elif [ -f prisma/schema.prisma ]; then npx --no-install prisma migrate deploy
  elif ls drizzle.config.* >/dev/null 2>&1; then npx --no-install drizzle-kit migrate
  elif ls knexfile.* >/dev/null 2>&1; then npx --no-install knex migrate:latest
  elif [ -f manage.py ]; then python3 manage.py migrate --noinput
  elif ls migrations/*.sql >/dev/null 2>&1; then
    # Plain SQL files, applied once each in name order and recorded in schema_migrations.
    psql_ "$db" -c 'create table if not exists schema_migrations (name text primary key, applied_at timestamptz default now())'
    for f in $(ls migrations/*.sql | sort); do
      n=$(basename "$f")
      [ -n "$(psql_ "$db" -tAc "select 1 from schema_migrations where name='$n'")" ] && continue
      { echo 'begin;'; cat "$f"; echo; echo "insert into schema_migrations(name) values ('$n');"; echo 'commit;'; } | psql_ "$db"
      echo "applied $n"
    done
  else
    echo "no migrations yet"
  fi
}

seed() {
  local db=${1:-$MAIN}; valid "$db"
  export DATABASE_URL; DATABASE_URL=$(url "$db")
  if [ -s scripts/seed.sh ]; then bash scripts/seed.sh; else echo "no seed script yet"; fi
}

cmd=${1:-status}; shift || true
case $cmd in
  ensure)
    [ -f docker-compose.yml ] || { echo "no docker-compose.yml"; exit 0; }
    start_docker
    docker compose up -d --wait >/dev/null 2>&1 || docker compose up -d
    wait_db
    for d in "$MAIN" "${MAIN}_test"; do
      [ -n "$(psql_ postgres -tAc "select 1 from pg_database where datname='$d'")" ] || psql_ postgres -c "create database \"$d\""
    done
    echo "services up: $(docker compose ps --services --status running | tr '\n' ' ')" ;;
  wait) wait_db ;;
  url) valid "$1"; url "$1" ;;
  create)
    valid "$1"
    [ -n "$(psql_ postgres -tAc "select 1 from pg_database where datname='$1'")" ] || psql_ postgres -c "create database \"$1\"" ;;
  drop)
    valid "$1"
    if [ "$1" = "$MAIN" ] && [ "$2" != --force ]; then echo "refusing to drop the main database $MAIN (add --force)"; exit 1; fi
    psql_ postgres -c "drop database if exists \"$1\" with (force)" ;;
  migrate) migrate "$1" ;;
  seed) seed "$1" ;;
  fresh)
    valid "$1"; [ "$1" = "$MAIN" ] && { echo "fresh is for scratch databases, use reset for $MAIN"; exit 1; }
    "$0" drop "$1"; "$0" create "$1"; migrate "$1" ;;
  reset)
    db=${1:-$MAIN}; valid "$db"
    "$0" drop "$db" --force; "$0" create "$db"; migrate "$db"; seed "$db" ;;
  sql) if [ $# -ge 2 ]; then valid "$1"; psql_ "$1" -c "$2"; else psql_ "$MAIN" -c "$1"; fi ;;
  psql) db=${1:-$MAIN}; valid "$db"
    if have_compose_db; then docker compose exec db psql -U "$U" -d "$db"; else PGPASSWORD=$PW psql -h localhost -p "$PORT" -U "$U" -d "$db"; fi ;;
  status)
    have_compose_db && docker compose ps --format '{{.Service}}: {{.State}}' || echo "db: not running (bash scripts/db.sh ensure)"
    psql_ postgres -tAc "select datname from pg_database where not datistemplate order by 1" 2>/dev/null | sed 's/^/database: /' || true ;;
  *) sed -n '2,16p' "$0"; exit 2 ;;
esac
