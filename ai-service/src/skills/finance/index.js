import { readFile } from 'node:fs/promises';

import { answerWithContext } from '../../rag/answer.js';
import { createRetriever } from '../../rag/retrieve.js';

export async function createFinanceSkill({ generate } = {}) {
  const raw = await readFile(new URL('../../../knowledge/finance.json', import.meta.url), 'utf8');
  const retrieve = createRetriever(JSON.parse(raw));

  return async ({ message, signal }) => {
    const question = message.trim();
    const documents = retrieve(question);
    return answerWithContext({ question, documents, generate, signal });
  };
}
