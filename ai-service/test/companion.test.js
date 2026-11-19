import test from 'node:test';
import assert from 'node:assert/strict';
import { createCompletion } from '../src/providers/chat-completions.js';
import { createCompanion } from '../src/skills/companion.js';

test('Ollama receives context and returns a function proposal', async () => {
  const complete = createCompletion({ url: 'http://127.0.0.1:11434/v1/chat/completions', model: 'test' },
    async (url, request) => {
      const { messages, tools } = JSON.parse(request.body);
      assert.equal(url.hostname, '127.0.0.1');
      assert.equal(messages[1].content, 'Aku mau beli kado 50k.');
      assert.equal(messages.at(-1).role, 'user');
      assert.ok(tools.some(t => t.function.name === 'simulate_purchase'));
      return Response.json({ choices: [{ message: { tool_calls: [{ type: 'function',
        function: { name: 'simulate_purchase', arguments: '{"amounts":[50000,25000]}' },
      }] } }] });
    });
  const plan = await createCompanion(complete)({ message: 'Kalau makan 25k juga?',
    history: [{ role: 'user', content: 'Aku mau beli kado 50k.' }] });
  assert.deepEqual(plan, { tool: 'simulate_purchase', arguments: { amounts: [50000, 25000] } });
});

test('companion answers curhat and rejects unsupported tool or history roles', async () => {
  const companion = createCompanion(async ({ tools }) => {
    assert.equal(tools, undefined);
    return { content: 'Ceritakan saja.' };
  });
  assert.deepEqual(await companion({ message: 'Aku cuma mau cerita.' }), { answer: 'Ceritakan saja.' });
  await assert.rejects(companion({ message: 'halo', history: [{ role: 'system', content: 'x' }] }));
  await assert.rejects(createCompanion(async () => ({
    tool_calls: [{ function: { name: 'unknown', arguments: '{}' } }],
  }))({ message: 'halo' }));
  assert.throws(() => createCompletion({ url: 'http://example.org/v1/chat/completions', model: 'test' }));
});
