import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'playwright');
const browser=await chromium.launch({headless:true});
const output=fileURLToPath(new URL('./',import.meta.url));
try{
 const page=await browser.newPage({viewport:{width:1536,height:1024}});
 const requests=[],errors=[];
 page.on('request',req=>requests.push(req.url()));page.on('pageerror',e=>errors.push(e.message));
 await page.goto(process.env.PREVIEW_URL??'http://127.0.0.1:5174');
 await page.locator('.home-office img').evaluate(img=>img.decode());
 assert.equal(await page.locator('video').count(),0);
 assert.equal(await page.locator('.home-office img').getAttribute('src'),'/images/control-room-reference.png');
 assert.ok(Math.abs(await page.evaluate(()=>document.querySelector('.home-office').getBoundingClientRect().width/document.querySelector('.home-center').getBoundingClientRect().width)-0.42)<0.002);
 await page.screenshot({path:`${output}desktop${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 for(const width of [1536,768,390,320]){
  await page.setViewportSize({width,height:1024});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
 }
 await page.screenshot({path:`${output}mobile${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.setViewportSize({width:1536,height:1024});
 await page.evaluate(()=>{window.sidebar=document.querySelector('.home-sidebar');window.topbar=document.querySelector('.home-topbar');});
 await page.getByRole('button',{name:'Configurações',exact:true}).click();
 await page.getByLabel('Token administrativo').waitFor();
 assert.equal(await page.evaluate(()=>window.sidebar===document.querySelector('.home-sidebar')&&window.topbar===document.querySelector('.home-topbar')),true);
 await page.getByRole('button',{name:'Voltar ao painel inicial',exact:true}).click();
 await page.locator('.home-office img').evaluate(img=>img.decode());
 assert.equal(await page.locator('video').count(),0);
 assert.ok(requests.every(url=>!url.includes('.mp4')&&!url.includes('control-room-poster')&&!url.includes('/api/')));
 assert.deepEqual(errors,[]);
 const result='PASS: original image decoded; no video element/media requests; desktop width 42% preserved; no overflow at 1536/768/390/320px; template/login/home navigation preserved; no page errors or API calls.\n';
 await writeFile(`${output}browser-results${process.env.RESULT_SUFFIX??''}.txt`,result);console.log(result);
}finally{await browser.close();}
