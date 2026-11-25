import { chromium } from 'playwright';
// usage: (cd site && python3 -m http.server 8765 --bind 127.0.0.1) &  then  npx -y -p playwright node e2e/site-check.mjs
const R=new URL('../reports/screenshots/', import.meta.url).pathname;
const url='http://127.0.0.1:8765/';
const b=await chromium.launch();
const out=(n,ok,d='')=>console.log(ok?'PASS':'FAIL',n,d);
for (const [name,vp] of [['desktop',{width:1280,height:800}],['mobile',{width:390,height:844}]]) {
  const p=await b.newPage({viewport:vp}); const errs=[];
  p.on('console',m=>{if(m.type()==='error')errs.push(m.text())}); p.on('pageerror',e=>errs.push(String(e)));
  await p.goto(url);
  if(name==='desktop'){
    out('title',(await p.title())==='AutoWiFi BITS');
    const beta=p.locator('.beta'); const t=await beta.innerText();
    out('beta visible',await beta.isVisible()&&t.includes('beta: not yet verified on the real BITS portal'));
    const box=await beta.boundingBox(); out('beta near top',box.y<300,'y='+box.y);
    const hrefs=await p.$$eval('a[href*="releases/download"]',a=>a.map(x=>x.href));
    const pre='https://github.com/AdityaEvilIncorporated/automaticwifi-bits/releases/download/v0.1.0/';
    out('7 download links',hrefs.length===7&&hrefs.every(h=>h.startsWith(pre)&&h.length>pre.length),hrefs.length);
  }
  const sw=await p.evaluate(()=>[document.documentElement.scrollWidth,innerWidth]);
  out(name+' no h-overflow',sw[0]<=sw[1],sw.join('/'));
  await p.screenshot({path:R+`site-${name}.png`,fullPage:true});
  if(name==='desktop'){
    await p.click('.beta a[href="#campus-checklist"]'); await p.waitForTimeout(500);
    const h=await p.evaluate(()=>{const s=document.getElementById('campus-checklist');const r=s.getBoundingClientRect();return [r.top,innerHeight,s.querySelector('h1,h2,h3')?.textContent]});
    out('anchor scrolls',h[0]>=-5&&h[0]<h[1]&&/Campus verification checklist/i.test(h[2]),JSON.stringify(h));
    await p.locator('#campus-checklist').screenshot({path:R+'site-checklist.png'});
  }
  out(name+' no console errors',errs.length===0,errs.join('|'));
  await p.close();
}
await b.close();
