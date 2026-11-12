import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'playwright');
const browser=await chromium.launch({headless:true});
const output=fileURLToPath(new URL('./',import.meta.url));
const timestamp='2026-10-06T01:00:00.000Z';
const roles=['PLANNER','DEVELOPER','REVIEWER','QA','DOCUMENTATION','SECURITY'];
const names=['Planner / Tech Lead','Developer','Reviewer','QA','Documentation','Security'];
let settings={version:1,createdAt:timestamp,updatedAt:timestamp,configuration:{installations:[],assignments:roles.map(role=>({role,enabled:false,installationId:null,model:null,permissionMode:'READ_ONLY',timeoutMinutes:10,maxAttempts:1})),github:{authMode:'GH_CLI',state:'DISCONNECTED',host:'github.com',owner:null,repository:null,baseBranch:'main',pullRequestCreationEnabled:false,mergeEnabled:false},financialSafety:{apiEnabled:false,extraUsageEnabled:false,paidCreditsEnabled:false,autoRechargeEnabled:false,paidFallbackEnabled:false}}};
let writes=0,conflict=false;
const results=[];
try{
 const page=await browser.newPage({viewport:{width:1536,height:1024}}),errors=[];
 page.on('pageerror',e=>errors.push(e.message));
 await page.route('**/api/**',route=>{
  const req=route.request(),path=new URL(req.url()).pathname;
  if(req.method()==='PUT'){
   assert.equal(path,'/api/settings');const body=req.postDataJSON();assert.equal(body.expectedVersion,settings.version);
   if(conflict)return route.fulfill({status:409,contentType:'application/json',body:'{"message":"Synthetic version conflict"}'});
   settings={...settings,version:settings.version+1,configuration:body.configuration};writes++;
  }else assert.equal(req.method(),'GET');
  return route.fulfill({status:path==='/api/operation'?503:200,contentType:'application/json',body:JSON.stringify(path==='/api/auth/session'?{authenticated:true}:path==='/api/settings'?settings:[])});
 });
 const open=async()=>{
  await page.goto(process.env.PREVIEW_URL??'http://127.0.0.1:5174');await page.getByRole('button',{name:'Configurações',exact:true}).click();
  if(await page.getByLabel('Token administrativo').count()){
   await page.getByLabel('Token administrativo').fill('synthetic-browser-test-credential-0000');await page.getByRole('button',{name:'Entrar',exact:true}).click();
  }
  await page.getByRole('tab',{name:'Equipes',exact:true}).click();await page.getByRole('switch',{name:'Ativar agente Developer',exact:true}).waitFor();
 };
 await open();
 assert.equal(await page.getByRole('switch').count(),6);assert.equal(await page.getByText('Funções de execução da fábrica',{exact:true}).count(),0);
 for(const name of names){
  await page.getByRole('button',{name:`Ver agente ${name}`,exact:true}).click();
  const modal=page.getByRole('dialog');await modal.getByRole('heading',{name,exact:true}).waitFor();
  assert.equal(await modal.getByRole('heading',{name:'Função do agente'}).count(),1);
  assert.equal(await modal.getByRole('heading',{name:'Participação no fluxo'}).count(),1);
  assert.equal(await modal.getByRole('button',{name:'Configurações do agente'}).count(),1);
  assert.equal(await modal.locator('input,select').count(),0);
  await page.keyboard.press('Escape');
 }
 assert.equal(writes,0);
 results.push('Six preset agents visible in Agents with individual switches; all six overview dialogs describe purpose/flow/current integration without writes or fields');
 const developer=page.getByRole('switch',{name:'Ativar agente Developer',exact:true});
 await developer.click();await page.waitForFunction(()=>document.querySelector('input[aria-label="Ativar agente Developer"]').getAttribute('aria-checked')==='true');
 assert.equal(settings.configuration.assignments.find(a=>a.role==='DEVELOPER').enabled,true);assert.equal(await page.getByRole('dialog').count(),0);
 const reviewer=page.getByRole('switch',{name:'Ativar agente Reviewer',exact:true});await reviewer.focus();await page.keyboard.press('Space');
 await page.waitForFunction(()=>document.querySelector('input[aria-label="Ativar agente Reviewer"]').getAttribute('aria-checked')==='true');
 await reviewer.click();await page.waitForFunction(()=>document.querySelector('input[aria-label="Ativar agente Reviewer"]').getAttribute('aria-checked')==='false');
 conflict=true;await developer.click();await page.getByRole('alert').waitFor();assert.equal(await developer.isChecked(),true);assert.equal(writes,3);conflict=false;
 results.push('Switch mouse/keyboard persist only selected role; 409 keeps stored checked state and shows list error, no modal opening');
 const view=page.getByRole('button',{name:'Ver agente Developer',exact:true});await view.click();
 await page.screenshot({path:`${output}overview${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.getByRole('button',{name:'Configurações do agente',exact:true}).click();
 await page.getByRole('dialog').getByRole('heading',{name:'Configurar Developer',exact:true}).waitFor();
 await page.getByLabel('Minutos',{exact:true}).fill('15');await page.getByLabel('Tentativas').selectOption('2');
 await page.screenshot({path:`${output}configuration${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.getByRole('button',{name:'Salvar configuração',exact:true}).click();await page.getByRole('dialog').waitFor({state:'hidden'});
 assert.equal(settings.configuration.assignments.find(a=>a.role==='DEVELOPER').timeoutMinutes,15);assert.equal(settings.configuration.assignments.find(a=>a.role==='DEVELOPER').maxAttempts,2);
 assert.equal(await view.evaluate(el=>el===document.activeElement),true);
 await open();assert.equal(await page.getByRole('switch',{name:'Ativar agente Developer',exact:true}).isChecked(),true);
 await page.getByRole('button',{name:'Ver agente Developer',exact:true}).click();await page.getByRole('button',{name:'Configurações do agente',exact:true}).click();assert.equal(await page.getByLabel('Minutos',{exact:true}).inputValue(),'15');
 await page.getByLabel('Minutos',{exact:true}).fill('20');await page.getByRole('button',{name:'Cancelar',exact:true}).click();assert.equal(settings.configuration.assignments.find(a=>a.role==='DEVELOPER').timeoutMinutes,15);
 results.push('Overview button opens existing behavior configuration; timeout/attempts persist after reload; cancel discards draft and focus returns to agent row');
 await page.getByRole('button',{name:'+ Adicionar agente',exact:true}).click();await page.getByLabel('Nome do agente',{exact:true}).fill('Personalizado');await page.getByRole('button',{name:'Salvar configuração',exact:true}).click();await page.getByRole('dialog').waitFor({state:'hidden'});
 assert.equal(await page.getByRole('button',{name:'Editar agente Personalizado',exact:true}).count(),1);assert.equal(await page.getByRole('switch').count(),6);
 await page.getByRole('tab',{name:'Skills',exact:true}).click();await page.getByRole('tab',{name:'Agentes',exact:true}).click();
 await page.screenshot({path:`${output}agents-desktop${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 for(const width of [1536,1024,768,390,320]){
  await page.setViewportSize({width,height:1024});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${width}`);
 }
 await page.screenshot({path:`${output}agents-mobile${process.env.RESULT_SUFFIX??''}.png`,fullPage:true});
 await page.getByRole('button',{name:'Ver agente Security',exact:true}).click();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
 assert.equal(await page.locator('.agent-overview h2').evaluate(el=>getComputedStyle(el).fontSize),'16px');await page.keyboard.press('Escape');
 assert.deepEqual(errors,[]);assert.equal(writes,5);
 results.push('Custom agents and Skills tabs preserved, compact 16px modal title, no overflow at five widths, no page errors');
 results.push('Five PUTs to in-memory versioned fixture only; no database/provider/authentication actions');
 await writeFile(`${output}browser-results${process.env.RESULT_SUFFIX??''}.txt`,results.join('\n')+'\n');console.log(results.join('\n'));
}finally{await browser.close();}
