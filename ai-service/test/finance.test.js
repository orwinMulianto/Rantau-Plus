import assert from 'node:assert/strict';
import { once } from 'node:events';
import test from 'node:test';

import { createAIService } from '../src/http.js';
import { createGenerator } from '../src/providers/chat-completions.js';
import { createFinanceSkill } from '../src/skills/finance/index.js';

test('finance menemukan panduan yang paling relevan dan menyertakan sumber', async () => {
  const finance = await createFinanceSkill();
  const result = await finance({ message: 'Apa itu Safe-to-Spend?' });

  assert.equal(result.mode, 'retrieval');
  assert.equal(result.sources[0].id, 'finance.safe-to-spend');
  assert.match(result.answer, /estimasi pengeluaran per hari/);
});

test('pertanyaan tanpa konteks tidak dikirim ke model', async () => {
  const generate = () => assert.fail('model tidak seharusnya dipanggil');
  const finance = await createFinanceSkill({ generate });
  const result = await finance({ message: 'Apa itu fotosintesis?' });

  assert.equal(result.mode, 'no_context');
  assert.deepEqual(result.sources, []);
});

test('RAG mengirim hasil retrieval ke adapter model', async () => {
  const generate = createGenerator(
    { url: 'https://provider.example/v1/chat/completions', model: 'test-model' },
    async (url, request) => {
      assert.equal(url.pathname, '/v1/chat/completions');
      const body = JSON.parse(request.body);
      const prompt = JSON.parse(body.messages[1].content);
      assert.equal(body.model, 'test-model');
      assert.equal(prompt.context[0].id, 'finance.runway');
      assert.equal(prompt.question, 'Jelaskan Financial Runway');
      return Response.json({ choices: [{ message: { content: 'Perkiraan ketahanan dana.' } }] });
    },
  );
  const finance = await createFinanceSkill({ generate });
  const result = await finance({ message: 'Jelaskan Financial Runway' });

  assert.equal(result.mode, 'rag');
  assert.equal(result.answer, 'Perkiraan ketahanan dana.');
  assert.equal(result.sources[0].id, 'finance.runway');
});

test('kegagalan model menampilkan kutipan dokumen', async () => {
  const finance = await createFinanceSkill({
    generate: async () => { throw new Error('timeout'); },
  });
  const result = await finance({ message: 'Financial Runway' });

  assert.equal(result.mode, 'retrieval');
  assert.equal(result.sources[0].id, 'finance.runway');
});

test('HTTP merutekan finance dan menolak skill yang belum terdaftar', async (t) => {
  const finance = await createFinanceSkill();
  const server = createAIService(new Map([['finance', finance]]));
  t.after(() => server.close());
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const endpoint = `http://127.0.0.1:${server.address().port}/internal/v1/chat`;

  async function chat(payload) {
    return fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  const ok = await chat({ message: 'Apa itu Financial Runway?' });
  assert.equal(ok.status, 200);
  assert.equal((await ok.json()).data.skill, 'finance');

  const unknown = await chat({ skill: 'other', message: 'Pertanyaan' });
  assert.equal(unknown.status, 400);
  assert.equal((await unknown.json()).error.code, 'UNKNOWN_SKILL');
});
