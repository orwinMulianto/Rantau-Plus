import { createServer } from 'node:http';

const maxBodyBytes = 8192;
const maxMessageLength = 1000;

function send(response, status, payload) {
  response.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  response.end(JSON.stringify(payload));
}

function fail(response, status, code, message) {
  send(response, status, { error: { code, message } });
}

export function createAIService(skills, companion) {
  return createServer(async (request, response) => {
    if (request.method === 'GET' && request.url === '/healthz') {
      send(response, 200, { data: { status: 'ok' } });
      return;
    }
    const isCompanion = request.url === '/internal/v1/companion';
    if (request.method !== 'POST' || (!isCompanion && request.url !== '/internal/v1/chat')) {
      fail(response, 404, 'NOT_FOUND', 'Endpoint tidak ditemukan.');
      return;
    }

    const contentType = request.headers['content-type']?.split(';', 1)[0].trim().toLowerCase();
    if (contentType !== 'application/json') {
      request.resume();
      fail(response, 415, 'UNSUPPORTED_MEDIA_TYPE', 'Gunakan format application/json.');
      return;
    }

    let body;
    try {
      const chunks = [];
      let bytes = 0;
      for await (const chunk of request.iterator({ destroyOnReturn: false })) {
        bytes += chunk.length;
        if (bytes > (isCompanion ? 64 * 1024 : maxBodyBytes)) {
          request.resume();
          fail(response, 413, 'REQUEST_TOO_LARGE', 'Permintaan terlalu besar.');
          return;
        }
        chunks.push(chunk);
      }
      body = JSON.parse(Buffer.concat(chunks).toString('utf8'));
    } catch {
      fail(response, 400, 'INVALID_JSON', 'Format JSON tidak valid.');
      return;
    }

    const { skill = 'finance', message } = body ?? {};
    if (typeof message !== 'string' || !message.trim() || [...message].length > maxMessageLength) {
      fail(response, 400, 'INVALID_MESSAGE', 'Isi pesan dengan 1–1000 karakter.');
      return;
    }
    if (isCompanion && !companion) {
      fail(response, 503, 'COMPANION_DISABLED', 'Model percakapan belum diaktifkan.');
      return;
    }
    const runSkill = isCompanion ? companion : skills.get(skill);
    if (!runSkill) {
      fail(response, 400, 'UNKNOWN_SKILL', 'Skill belum tersedia.');
      return;
    }

    const controller = new AbortController();
    response.once('close', () => controller.abort());
    try {
      const result = await runSkill({ message, history: body.history, hasLocation: body.has_location === true, signal: controller.signal });
      send(response, 200, { data: isCompanion ? result : { skill, ...result } });
    } catch {
      fail(response, 500, 'SKILL_ERROR', 'Skill gagal memproses pesan.');
    }
  });
}
