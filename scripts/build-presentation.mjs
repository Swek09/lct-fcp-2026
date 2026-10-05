// Run a copy from output/presentation-build with a node_modules link to the
// bundled Codex runtime. See docs/development.md for the complete command.
import fs from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {Presentation, PresentationFile} from '@oai/artifact-tool';

const workspaceDir = process.env.WORKSPACE_DIR;
const skillDir = process.env.SKILL_DIR;
const runtimePython = process.env.RUNTIME_PYTHON;
if (![workspaceDir, skillDir, runtimePython].every(v=>v && path.isAbsolute(v))) {
  throw new Error('WORKSPACE_DIR, SKILL_DIR and RUNTIME_PYTHON must be absolute paths');
}
const {resolvePresentationFont, applyPresentationChartFont, finalizePresentation} = await import(pathToFileURL(path.join(skillDir, 'container_tools/artifact_tool_utils.mjs')).href);
const family = resolvePresentationFont({fontFamily:'Arial', availableFonts:['Arial']});
const buildDir = path.join(workspaceDir,'output','presentation-build');
const finalPath = process.env.FINAL_PPTX || path.join(workspaceDir,'output','presentations','fsp-career-final.pptx');
await fs.mkdir(buildDir,{recursive:true});
await fs.mkdir(path.dirname(finalPath),{recursive:true});
const p = Presentation.create({slideSize:{width:1280,height:720}});
const bg='#11181C', light='#F1F5F7', mute='#B0BDC5', blue='#7496FF';
function text(s,value,x,y,w,h,size=28,color=light,bold=false){
  const t=s.shapes.add({geometry:'textbox',position:{left:x,top:y,width:w,height:h},fill:'none',line:{fill:'none',width:0}});
  t.text=value;
  t.text.style={typeface:family,fontSize:size,bold,color,autoFit:'none'};
  return t;
}
function slide(title,notes){
  const s=p.slides.add(); s.background.fill=bg;
  if(title)text(s,title,70,48,1136,90,44,light,true);
  s.speakerNotes.textFrame.setText(notes);
  return s;
}
function row(s,n,title,body,y){
  text(s,n,72,y,78,68,42,blue,true);
  text(s,title,177,y,1000,48,30,light,true);
  text(s,body,177,y+57,1000,82,26,mute);
}
let s=slide('', 'Источник: docs/specification.md, README.md. Локальный прототип MVP. Данные профилей и достижений синтетические.');
text(s,'ФСП',72,76,400,65,40,blue,true);
text(s,'Карьера',72,205,1035,130,94,light,true);
text(s,'Работодатель начинает разговор',76,367,1100,82,40,light);
text(s,'Платформа подбора ИТ-специалистов\nпо подтверждённой категории',76,480,1030,105,29,mute);
text(s,'Прототип / октябрь 2026',76,650,900,28,20,mute);

s=slide('Механика обратного найма','Источник: docs/architecture.md, docs/demo.md и реализация приглашений в backend/internal/app/invitations.go. Принятие раскрывает email только отправителю принятого приглашения.');
row(s,'01','Кандидат подтверждает уровень','Профиль и тест определяют категорию, доступную работодателю.',166);
row(s,'02','Работодатель выбирает человека','Поиск объясняет совпадение. Предложение сразу содержит зарплату.',329);
row(s,'03','Кандидат разрешает контакт','Принятие приглашения открывает электронную почту этой компании.',492);

s=slide('Подтверждение категории','Источник: docs/architecture.md, backend/internal/assessment/engine.go и scenarios.go. Порог 70 баллов требует минимум 5 из 6 ответов. Повтор той же категории через 24 часа, смена подтверждённой категории через 90 дней.');
text(s,'6 заданий',72,169,600,83,64,blue,true);
text(s,'20 минут на попытку',72,267,690,55,34,light);
text(s,'5 правильных ответов для категории',72,339,1100,60,34,light);
text(s,'Бэкенд, фронтенд, аналитика\nJunior, Middle, Senior',72,449,1070,99,30,mute);
text(s,'При неудаче прежний грейд сохраняется.\nСмена подтверждённой категории доступна через 90 дней.',72,575,1130,94,26,mute);

s=slide('Объяснение подбора','Источник: backend/internal/matching/rank.go, docs/architecture.md. Веса: тест 50, совпадение стека 40, ФСП 10. Достижения демонстрационные, бонус ограничен тремя. Стек в профиле самоописан, тест подтверждает общую специализацию и грейд. Без стека балл нормируется.');
text(s,'Максимальный вклад в балл соответствия',72,144,1110,42,27,mute);
const chart=s.charts.add('bar',{
  position:{left:72,top:206,width:725,height:332},
  categories:['Результат теста','Совпадение стека','Достижения ФСП'],
  series:[{name:'Баллы',values:[50,40,10],fill:'#7496FF'}],
  barOptions:{direction:'bar',grouping:'clustered',gapWidth:80},
  hasLegend:false, dataLabels:{showValue:true,position:'outEnd',textStyle:{fontFamily:family,fontSize:26,fill:light,bold:true}},
  chartFill:bg,plotAreaFill:bg,
});
applyPresentationChartFont(chart,{fontFamily:family});
chart.xAxis={textStyle:{fontFamily:family,fontSize:22,fill:light},line:{fill:'none',width:0}};
chart.yAxis={visible:false,textStyle:{fontFamily:family,fontSize:24,fill:light},majorGridlines:null,min:0,max:60};
text(s,'ФСП даёт бонус.\nКандидат без истории\nостаётся в подборке.',859,234,339,198,30,light);
text(s,'Подтверждённая категория задаёт фильтр. Навыки профиля уточняют совпадение.',72,595,1120,76,26,mute);

s=slide('Рабочий прототип','Источник изображения: собственный скриншот приложения output/playwright/employer-desktop.png. Данные синтетические. Локальное приложение http://localhost:8080.');
const screenshot=await fs.readFile(path.join(workspaceDir,'output','playwright','employer-desktop.png'));
s.images.add({blob:new Uint8Array(screenshot),contentType:'image/png',alt:'Поиск кандидатов в кабинете работодателя',fit:'contain',position:{left:71,top:152,width:1138,height:490}});
text(s,'Два кабинета. Поиск, профиль и приглашения с зарплатой.',72,661,1110,30,23,mute);

const report=JSON.parse(await fs.readFile(path.join(workspaceDir,'docs','assessment-validation.json'),'utf8'));
const variants=report.results.reduce((a,r)=>a+r.variants,0);
const copied=report.results.reduce((a,r)=>a+r.copied_key_passes,0);
s=slide('Проверка вариантов тестов','Источник: docs/assessment-validation.json, cmd/evaluate. По 100 новых seed=2..101 на девять категорий. Копирование ключа seed=1 без решения новых условий. Это механическая синтетическая проверка, не оценка реальных грейдов или устойчивости к помощи LLM.');
text(s,`${(copied/variants*100).toFixed(1).replace('.',',')}%`,72,170,1050,136,100,blue,true);
text(s,`${copied} из ${variants} новых тестов прошли\nс готовым ключом от другого варианта`,76,331,1100,126,38,light);
text(s,'Условия меняются при сохранении общего плана категории.',76,493,1100,55,29,mute);
text(s,'Точность грейдов и сопоставимость сложности\nещё требуют проверки экспертами и кандидатами.',76,588,1100,89,27,mute);

s=slide('Стек и хранение','Источник: backend/go.mod, frontend/package.json, compose.yaml и docs/architecture.md. Redis хранит сессии и счётчики. PostgreSQL хранит предметные данные, дедлайны и правила смены категории. Модульный монолит.');
text(s,'React + TypeScript',72,169,1090,65,43,blue,true);
text(s,'Vite, Tailwind, TanStack Query',72,245,1090,50,29,mute);
text(s,'Go + chi + Huma',72,340,1090,65,43,light,true);
text(s,'Модульный монолит, OpenAPI, SMTP outbox',72,417,1090,50,29,mute);
text(s,'PostgreSQL + Redis',72,506,1090,65,43,light,true);
text(s,'Результаты и правила в БД. Сессии и лимиты в Redis.',72,583,1090,55,29,mute);

s=slide('Следующий этап','Источник: docs/architecture.md и ограничения README.md. Реальный API ФСП отсутствует в исходном ТЗ. OIDC/Keycloak, ATS и проверки работодателей являются предложенной концепцией. Нет подключённого внешнего LLM.');
row(s,'01','Экспертная калибровка','Расширение банка, независимая разметка грейдов и качества подбора.',158);
row(s,'02','Интеграции','ФСП ID через OIDC, контракт достижений и события для ATS.',321);
row(s,'03','Внешний стенд','Домен, HTTPS, почтовый сервис и база без демоаккаунтов.',484);

const candidatePath=path.join(buildDir,'candidate.pptx');
await (await PresentationFile.exportPptx(p)).save(candidatePath);
const result=await finalizePresentation({
  workspaceDir,candidatePath,finalPath,pythonExecutable:runtimePython,
  integrityValidatorPath:path.join(skillDir,'container_tools/inspect_presentation_package_integrity.py'),
  layoutValidatorPath:path.join(skillDir,'container_tools/inspect_presentation_layout_geometry.py'),
  layoutArgs:['--expected-slide-size-emu','12192000,6858000','--validate-heading-fit'],
  fontPolicy:{basis:'design',families:[family]},
  requiredNativeChartOwnerSlides:[4],requiredNativeTableOwnerSlides:[],
  materializeLiteralChartWorkbooks:true,
  verifyArtifactToolImport:true,
  receiptPath:path.join(buildDir,path.basename(finalPath)+'.validation.json'),
});
for(let i=0;i<p.slides.items.length;i++){
  const slide=p.slides.items[i];
  const render=await p.export({slide,format:'png',scale:1});
  await fs.writeFile(path.join(buildDir,`slide-${i+1}.png`),new Uint8Array(await render.arrayBuffer()));
}
console.log(JSON.stringify(result));
