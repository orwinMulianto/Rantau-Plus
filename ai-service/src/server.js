import { createAIService } from './http.js';
import { createGenerator } from './providers/chat-completions.js';
import { createFinanceSkill } from './skills/finance/index.js';

const port = Number(process.env.AI_SERVICE_PORT ?? 8081);
if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error('AI_SERVICE_PORT harus berupa nomor port yang valid.');
}

const generate = createGenerator({
  url: process.env.AI_LLM_URL,
  model: process.env.AI_LLM_MODEL,
  apiKey: process.env.AI_LLM_API_KEY,
});
const finance = await createFinanceSkill({ generate });
const server = createAIService(new Map([['finance', finance]]));

server.listen(port, '127.0.0.1', () => {
  console.log(`Rantau+ AI Service berjalan pada 127.0.0.1:${port}`);
});
