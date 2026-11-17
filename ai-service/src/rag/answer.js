export async function answerWithContext({ question, documents, generate, signal }) {
  if (documents.length === 0) {
    return {
      mode: 'no_context',
      answer: 'Saya belum menemukan topik itu di panduan Rantau+.',
      sources: [],
    };
  }

  const sources = documents.map(({ id, title }) => ({ id, title }));
  if (generate) {
    try {
      const answer = await generate({ question, documents, signal });
      return { mode: 'rag', answer, sources };
    } catch {
      // The retrieved guide still answers supported questions if generation fails.
    }
  }

  return {
    mode: 'retrieval',
    answer: documents.map(({ title, text }) => `${title}\n${text}`).join('\n\n'),
    sources,
  };
}
