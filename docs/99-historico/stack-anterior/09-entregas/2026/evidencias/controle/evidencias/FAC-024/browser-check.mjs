import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'playwright');
const browser=await chromium.launch({headless:true});
const output=fileURLToPath(new URL('./',import.meta.url));
const results=[];
try{
 const page=await browser.newPage({viewport:{width:1536,height:1024}});
 const errors=[],api=[];
 page.on('pageerror',e=>errors.push(e.message));
 page.on('request',req=>{if(req.url().includes('/api/'))api.push(req.url());});
 await page.route('**/api/**',route=>route.fulfill({status:401,contentType:'application/json',body:'{"message":"Synthetic unauthenticated test"}'}));
 await page.goto(process.env.PREVIEW_URL??'http://127.0.0.1:5174');
 await page.waitForFunction(()=>{const v=document.querySelector('.home-office video');return v?.readyState===4&&!v.paused&&v.currentTime>0.1;});
 const attrs=await page.locator('video').evaluate(v=>({controls:v.controls,muted:v.muted,loop:v.loop,autoplay:v.autoplay,inline:v.playsInline,pip:v.disablePictureInPicture,remote:v.disableRemotePlayback,duration:v.duration,width:v.videoWidth,height:v.videoHeight,tabIndex:v.tabIndex,hidden:v.getAttribute('aria-hidden')}));
 assert.deepEqual({...attrs,duration:undefined},{controls:false,muted:true,loop:true,autoplay:true,inline:true,pip:true,remote:true,duration:undefined,width:960,height:540,tabIndex:-1,hidden:'true'});
 assert.ok(Math.abs(attrs.duration-9.25)<0.05);
 assert.equal(await page.locator('.home-office button').count(),0);
 assert.equal(await page.locator('video').evaluate(v=>!v.dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,cancelable:true}))),true);
 results.push('Real local MP4 autoplay, muted, loop, playsInline; no controls, PiP/remote disabled, no overlay buttons or keyboard focus');
 const ratio=await page.evaluate(()=>document.querySelector('.home-office').getBoundingClientRect().width/document.querySelector('.home-center').getBoundingClientRect().width);
 assert.ok(Math.abs(ratio-0.42)<0.002);
 results.push('Desktop width 42% of center = 70% of previous 60%; 16:9 video aspect ratio');
 const continuity=await page.locator('video').evaluate(v=>new Promise((resolve,reject)=>{
  const canvas=document.createElement('canvas');canvas.width=32;canvas.height=18;
  const ctx=canvas.getContext('2d');const samples=[];let wraps=0;
  const timeout=setTimeout(()=>reject(new Error('Loop observation timeout')),24000);
  const sample=(now,meta)=>{
   ctx.drawImage(v,0,0,32,18);
   const pixels=ctx.getImageData(0,0,32,18).data;
   let luminance=0;for(let i=0;i<pixels.length;i+=4)luminance+=(pixels[i]+pixels[i+1]+pixels[i+2])/3;
   const previous=samples.at(-1);
   if(previous&&meta.mediaTime<previous.time)wraps++;
   samples.push({time:meta.mediaTime,now,luminance:luminance/(32*18)});
   if(wraps===2){clearTimeout(timeout);resolve({wraps,frames:samples.length,minLuminance:Math.min(...samples.map(s=>s.luminance)),maxFrameGap:Math.max(...samples.slice(1).map((s,i)=>s.now-samples[i].now)),seamGaps:samples.slice(1).flatMap((s,i)=>s.time<samples[i].time?[s.now-samples[i].now]:[])});}
   else v.requestVideoFrameCallback(sample);
  };
  v.currentTime=v.duration-0.5;
  v.requestVideoFrameCallback(sample);
 }));
 assert.equal(continuity.wraps,2);assert.ok(continuity.minLuminance>10);assert.ok(continuity.maxFrameGap<350);assert.ok(continuity.seamGaps.every(g=>g<150));
 results.push(`Two native loops: ${continuity.frames} displayed frames, maximum gap ${continuity.maxFrameGap.toFixed(1)}ms, seam gaps ${continuity.seamGaps.map(g=>g.toFixed(1)).join('/')}ms, no black frame`);
 await page.screenshot({path:`${output}desktop.png`,fullPage:true});
 for(const width of [1536,1280,1024,768,390,320]){
  await page.setViewportSize({width,height:1024});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${width}`);
  if(width<760){const size=await page.evaluate(()=>({w:document.querySelector('.home-office').getBoundingClientRect().width,c:document.querySelector('.home-center').getBoundingClientRect().width}));assert.ok(Math.abs(size.w-(size.c-24)*0.7)<1);}
 }
 await page.screenshot({path:`${output}mobile.png`,fullPage:true});
 await page.setViewportSize({width:1536,height:1024});
 await page.evaluate(()=>{window.sidebar=document.querySelector('.home-sidebar');window.topbar=document.querySelector('.home-topbar');});
 assert.equal(api.length,0);
 await page.getByRole('button',{name:'Projetos',exact:true}).click();
 await page.getByLabel('Token administrativo').waitFor();
 assert.equal(await page.locator('video').count(),0);
 assert.equal(await page.evaluate(()=>window.sidebar===document.querySelector('.home-sidebar')&&window.topbar===document.querySelector('.home-topbar')),true);
 await page.getByRole('button',{name:'Voltar ao painel inicial',exact:true}).click();
 await page.waitForFunction(()=>{const v=document.querySelector('video');return v&&!v.paused&&v.currentTime>0.1;});
 assert.deepEqual(errors,[]);
 results.push('No overflow at six widths; mobile at 70% of former width; auth/navigation/frame preserved, video unmounts on other pages and restarts on home');
 results.push('No page errors or initial API calls; browser media measurements on local asset, no provider/database actions');
 await writeFile(`${output}browser-results${process.env.RESULT_SUFFIX??''}.txt`,results.join('\n')+'\n');
 console.log(results.join('\n'));
}finally{await browser.close();}
