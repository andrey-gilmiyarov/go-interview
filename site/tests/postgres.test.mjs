import assert from 'node:assert/strict'
import { test, afterEach } from 'node:test'
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import * as api from '../scripts/content.mjs'
const roots=[]
afterEach(()=>roots.splice(0).forEach(root=>rmSync(root,{recursive:true,force:true})))
function fixture(records){const root=mkdtempSync(path.join(os.tmpdir(),'postgres-content-'));roots.push(root);mkdirSync(path.join(root,'site/data'),{recursive:true});mkdirSync(path.join(root,'site/postgres'),{recursive:true});writeFileSync(path.join(root,'site/data/groups.json'),JSON.stringify(api.GROUP_IDS.map(id=>({id,title:id}))));if(records!==undefined)writeFileSync(path.join(root,'site/data/postgres.json'),JSON.stringify(records));for(const r of records??[]){if(r?.id && !r.id.includes('/'))writeFileSync(path.join(root,'site/postgres',r.id+'.md'),api.REQUIRED_HEADINGS.map(h=>'## '+h).join('\n'))}return root}
const record=(overrides={})=>({id:'storage-mvcc',title:'MVCC',summary:'Версии строк',order:1,postgresVersion:'17.11',reviewedAt:'2026-10-03',sources:['https://www.postgresql.org/docs/17/mvcc.html'],...overrides})
test('Postgres loader and sidebar preserve existing registry interfaces',()=>{assert.equal(typeof api.loadPostgresRegistry,'function');const root=fixture([record({order:2}),record({id:'locks',order:1})]);assert.deepEqual(api.loadPostgresRegistry({root}).map(x=>x.id),['storage-mvcc','locks']);assert.deepEqual(api.buildPostgresSidebar({root}).items.map(x=>x.link),['/postgres/','/postgres/locks','/postgres/storage-mvcc']);assert.deepEqual(Object.keys(api.loadRegistry({root})),['groups','topics','exercises'])})
test('missing registry allowed only in partial mode',()=>{const root=fixture();assert.deepEqual(api.loadPostgresRegistry({root}),[]);assert.deepEqual(api.checkContent({root,partial:true}).errors,[]);assert.match(api.checkContent({root}).errors.join('\n'),/Expected 8 PostgreSQL topics/);assert.equal(api.checkContent({root,partial:true}).stats.postgresTopics,0)})
test('metadata rejects duplicates, bad versions, invalid dates and sources',()=>{const root=fixture([record(),record({postgresVersion:'17.11.0',reviewedAt:'2026-02-30',sources:['http://example.org']})]);const errors=api.checkContent({root,partial:true}).errors.join('\n');for(const pattern of [/duplicate PostgreSQL topic id/,/duplicate PostgreSQL topic order/,/postgresVersion/,/reviewedAt/,/HTTPS/])assert.match(errors,pattern)})
test('lab links are checked and nested solutions stay out of search',()=>{const root=fixture([record()]);mkdirSync(path.join(root,'labs/postgres'),{recursive:true});writeFileSync(path.join(root,'labs/postgres/README.md'),'[broken](missing.md)');assert.match(api.checkContent({root,partial:true}).errors.join('\n'),/missing.md/);assert.equal(api.stripNonSearchableContent('<p>answer</p>','postgres/solutions/stock-race.md'),'')})

test('repository has eight chapters and eight separate non-searchable solutions',async()=>{
 const {readFileSync,existsSync}=await import('node:fs')
 const records=api.loadPostgresRegistry()
 assert.deepEqual(records.map(x=>x.id),['storage-mvcc','transactions-isolation','locks','indexes','query-plans','schema-integrity','go-access','operations-migrations'])
 const ids=['stock-race','isolation','deadlock','idempotency','query-plan','job-queue','long-transaction','go-pool']
 for(const id of ids){
  const solution=readFileSync(new URL('../postgres/solutions/'+id+'.md',import.meta.url),'utf8')
  assert.match(solution,/search: false/)
  assert.match(solution,new RegExp('practice/'+id+'/SOLUTION.md'))
  assert.ok(existsSync(new URL('../../labs/postgres/practice/'+id+'/README.md',import.meta.url)))
 }
 assert.equal(api.checkContent().stats.postgresTopics,8)
})
test('malformed PostgreSQL registries and missing headings fail clearly',()=>{
 const root=fixture([record()]);const file=path.join(root,'site/data/postgres.json')
 writeFileSync(file,'{broken');assert.throws(()=>api.loadPostgresRegistry({root}),/invalid JSON/)
 writeFileSync(file,'{}');assert.throws(()=>api.loadPostgresRegistry({root}),/array/)
 writeFileSync(file,JSON.stringify([record()]));writeFileSync(path.join(root,'site/postgres/storage-mvcc.md'),'# Empty')
 assert.match(api.checkContent({root,partial:true}).errors.join('\n'),/missing required heading/)
})
