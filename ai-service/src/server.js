import { createAIService } from './http.js';
import { createCompletion, createGenerator } from './providers/chat-completions.js';
import { createCompanion } from './skills/companion.js';
import { createFinanceSkill } from './skills/finance/index.js';

try {
  process.loadEnvFile(new URL('../.env', import.meta.url));
} catch (err) {
  if (err.code !== 'ENOENT') throw err;
}
const port = Number(process.env.AI_SERVICE_PORT ?? 8081);
if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error('AI_SERVICE_PORT harus berupa nomor port yang valid.');
}

const config = {
  url: process.env.AI_LLM_URL,
  model: process.env.AI_LLM_MODEL,
  apiKey: process.env.AI_LLM_API_KEY,
};
const generate = createGenerator(config);
const finance = await createFinanceSkill({ generate });
const enabled = process.env.AI_COMPANION_ENABLED ?? 'false';
if (!['true', 'false'].includes(enabled)) throw new Error('AI_COMPANION_ENABLED harus true atau false.');
const companion = enabled === 'true' ? createCompanion(createCompletion(config)) : undefined;
const server = createAIService(new Map([['finance', finance]]), companion);

server.listen(port, '127.0.0.1', () => {
  console.log(`Rantau+ AI Service berjalan pada 127.0.0.1:${port}`);
});
