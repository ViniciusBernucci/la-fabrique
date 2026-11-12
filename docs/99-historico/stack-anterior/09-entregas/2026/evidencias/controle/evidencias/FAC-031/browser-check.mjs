import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'playwright');
const browser=await chromium.launch({headless:true});
const output=fileURLToPath(new URL('./',import.meta.url));
const origin=process.env.PREVIEW_URL??'http://127.0.0.1:5174';
try{
 const page=await browser.newPage({viewport:{width:1536,height:1024}}),errors=[];
 page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>localStorage.setItem('la-fabrique.sidebar-collapsed','false'));
 await page.goto(origin);await page.locator('.home-office img').evaluate(img=>img.decode());
 const width=()=>page.locator('.home-sidebar').evaluate(el=>el.getBoundingClientRect().width);
 assert.equal(await width(),66);
 const toggle=page.getByRole('button',{name:'Expandir menu',exact:true});
 assert.equal(await toggle.innerText(),'');
 assert.equal(await toggle.evaluate(el=>getComputedStyle(el).borderTopWidth),'0px');
 const geometry=async()=>page.evaluate(()=>{
  const menu=document.querySelector('.home-sidebar').getBoundingClientRect();
  const arrow=document.querySelector('.home-menu-toggle svg').getBoundingClientRect();
  return {centerOffset:(arrow.left+arrow.right-menu.left-menu.right)/2,rightGap:menu.right-arrow.right};
 });
 assert.ok(Math.abs((await geometry()).centerOffset)<=0.5);
 await page.screenshot({path:`${output}default${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await toggle.focus();await page.keyboard.press('Space');assert.equal(await width(),234);assert.equal((await geometry()).rightGap,13);await page.screenshot({path:`${output}expanded${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.getByRole('button',{name:'Configurações',exact:true}).click();await page.getByLabel('Token administrativo').waitFor();assert.equal(await width(),234);
 await page.getByRole('button',{name:'Voltar ao painel inicial',exact:true}).click();assert.equal(await width(),234);
 await page.reload();assert.equal(await width(),66);
 for(const w of [1536,1024,768,390,320]){await page.setViewportSize({width:w,height:1024});assert.equal(await width(),66);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);}
 await page.getByRole('button',{name:'Expandir menu',exact:true}).click();assert.equal(await width(),234);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await page.getByRole('button',{name:'Recolher menu',exact:true}).click();
 await page.screenshot({path:`${output}mobile${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 const blocked=await browser.newPage({viewport:{width:1536,height:1024}});await blocked.addInitScript(()=>Object.defineProperty(window,'localStorage',{get(){throw new Error('Synthetic disabled storage');}}));blocked.on('pageerror',e=>errors.push(e.message));await blocked.goto(origin);assert.equal(await blocked.locator('.home-sidebar').evaluate(el=>el.getBoundingClientRect().width),66);assert.deepEqual(errors,[]);
 const result='PASS: menu starts at 66px despite previous expanded preference or unavailable storage; arrow centered when collapsed and 12px inset when expanded, no visible text/border; keyboard expands to 234px, navigation keeps choice, reload restores 66px; five widths/mobile toggle without overflow or page errors.\n';
 await writeFile(`${output}browser-results${process.env.RESULT_SUFFIX??''}.txt`,result);console.log(result);
}finally{await browser.close();}
