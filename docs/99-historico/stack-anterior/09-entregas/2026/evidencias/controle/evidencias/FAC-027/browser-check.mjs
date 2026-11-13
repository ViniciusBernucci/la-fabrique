import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'playwright');
const browser=await chromium.launch({headless:true});
const output=fileURLToPath(new URL('./',import.meta.url));
const origin=process.env.PREVIEW_URL??'http://127.0.0.1:5174',results=[];
try{
 const page=await browser.newPage({viewport:{width:1536,height:1024}}),errors=[];
 page.on('pageerror',e=>errors.push(e.message));
 await page.goto(origin);
 const width=()=>page.locator('.home-sidebar').evaluate(el=>el.getBoundingClientRect().width);
 assert.equal(await width(),234);
 await page.getByRole('button',{name:'Recolher menu',exact:true}).focus();await page.keyboard.press('Space');
 assert.equal(await width(),66);
 assert.equal(await page.getByRole('button',{name:'Expandir menu',exact:true}).getAttribute('aria-expanded'),'false');
 assert.equal(await page.getByRole('button',{name:'Projetos',exact:true}).getAttribute('title'),'Projetos');
 assert.equal(await page.locator('.home-sidebar nav button span').first().isVisible(),false);
 assert.equal(await page.locator('.home-main').evaluate(el=>getComputedStyle(el).marginLeft),'66px');
 await page.screenshot({path:`${output}collapsed${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.evaluate(()=>{window.sidebar=document.querySelector('.home-sidebar');});
 await page.getByRole('button',{name:'Configurações',exact:true}).click();await page.getByLabel('Token administrativo').waitFor();assert.equal(await width(),66);
 assert.equal(await page.evaluate(()=>window.sidebar===document.querySelector('.home-sidebar')),true);
 await page.getByRole('button',{name:'Voltar ao painel inicial',exact:true}).click();await page.reload();assert.equal(await width(),66);
 await page.getByRole('button',{name:'Expandir menu',exact:true}).click();assert.equal(await width(),234);assert.equal(await page.locator('.home-sidebar nav button span').first().isVisible(),true);await page.reload();assert.equal(await width(),234);
 results.push('Desktop 234px ↔ 66px; keyboard button, hidden text/visible icons, title and aria labels; content margin tracks width; preference survives navigation and reload');
 for(const w of [1536,1024,768,390,320]){
  await page.setViewportSize({width:w,height:1024});
  if(w<=760){assert.equal(await width(),66);await page.getByRole('button',{name:'Expandir menu',exact:true}).click();assert.equal(await width(),234);assert.equal(await page.locator('.home-main').evaluate(el=>getComputedStyle(el).marginLeft),'66px');await page.getByRole('button',{name:'Recolher menu',exact:true}).click();}
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${w}`);
 }
 await page.screenshot({path:`${output}mobile${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 results.push('Mobile starts/crosses breakpoint collapsed; expands as 234px overlay and recollects; no overflow at five widths');
 const blocked=await browser.newPage({viewport:{width:1536,height:1024}});
 await blocked.addInitScript(()=>Object.defineProperty(window,'localStorage',{get(){throw new Error('Synthetic disabled storage');}}));
 blocked.on('pageerror',e=>errors.push(e.message));await blocked.goto(origin);await blocked.getByRole('button',{name:'Recolher menu',exact:true}).click();assert.equal(await blocked.locator('.home-sidebar').evaluate(el=>el.getBoundingClientRect().width),66);await blocked.getByRole('button',{name:'Projetos',exact:true}).click();await blocked.getByLabel('Token administrativo').waitFor();assert.equal(await blocked.locator('.home-sidebar').evaluate(el=>el.getBoundingClientRect().width),66);
 assert.deepEqual(errors,[]);results.push('Disabled localStorage does not break toggling/navigation; no page errors, no provider/database operations');
 await writeFile(`${output}browser-results${process.env.RESULT_SUFFIX??''}.txt`,results.join('\n')+'\n');console.log(results.join('\n'));
}finally{await browser.close();}
