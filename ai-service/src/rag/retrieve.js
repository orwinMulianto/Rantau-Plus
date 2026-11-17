const stopWords = new Set([
  'ada', 'adalah', 'aku', 'apa', 'atau', 'bagaimana', 'dan', 'dengan', 'di',
  'ini', 'itu', 'ke', 'kalau', 'maksudnya', 'menurut', 'saya', 'yang', 'untuk',
]);

function terms(text) {
  const words = text.toLowerCase().match(/[\p{L}\p{N}]+/gu) ?? [];
  return new Set(words.filter((word) => !stopWords.has(word)));
}

export function createRetriever(documents) {
  const index = documents.map((document) => ({
    document,
    title: terms(document.title),
    keywords: terms(document.keywords),
    body: terms(document.text),
  }));

  return (question, limit = 2) => {
    const query = terms(question);
    return index
      .map(({ document, title, keywords, body }) => {
        let score = 0;
        for (const term of query) {
          if (title.has(term)) score += 4;
          else if (keywords.has(term)) score += 2;
          else if (body.has(term)) score += 1;
        }
        return { document, score };
      })
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, limit)
      .map(({ document }) => document);
  };
}
