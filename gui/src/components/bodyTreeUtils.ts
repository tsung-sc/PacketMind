export const EMPTY_BODY_PLACEHOLDER = '(empty)';

export type BodyTreeKind = 'json' | 'xml';

export interface BodyTreeNode {
  id: string;
  label: string;
  value?: string;
  nodeType:
    | 'object'
    | 'array'
    | 'string'
    | 'number'
    | 'boolean'
    | 'null'
    | 'xml-element'
    | 'xml-attribute'
    | 'xml-text'
    | 'xml-cdata'
    | 'xml-comment';
  children?: BodyTreeNode[];
}

export interface StructuredBodyTree {
  kind: BodyTreeKind;
  root: BodyTreeNode;
}

const utf8Decoder = new TextDecoder('utf-8', { fatal: true });

const normalizeContentType = (contentType: string | null) => (contentType || '').toLowerCase();

const baseContentType = (contentType: string | null) => normalizeContentType(contentType).split(';')[0].trim();

const extractCharset = (contentType: string | null): string => {
  const normalized = normalizeContentType(contentType);
  const match = normalized.match(/charset=([\w-]+)/);
  return match ? match[1] : 'utf-8';
};

const looksLikeBase64 = (value: string) => {
  const trimmed = value.trim();
  if (trimmed.length === 0) return false;
  // Allow base64 with or without padding; check character set only
  // Valid base64: [A-Za-z0-9+/]+ optionally followed by up to 2 '=' chars
  return /^[A-Za-z0-9+/]+={0,2}$/.test(trimmed);
};

const bytesFromBase64 = (value: string): Uint8Array => {
  const decoded = atob(value.trim());
  const bytes = new Uint8Array(decoded.length);
  for (let index = 0; index < decoded.length; index += 1) {
    bytes[index] = decoded.charCodeAt(index);
  }
  return bytes;
};

const decodeTextBytes = (bytes: Uint8Array, charset: string = 'utf-8') => {
  try {
    const decoder = new TextDecoder(charset, { fatal: true });
    return decoder.decode(bytes);
  } catch {
    if (charset.toLowerCase() !== 'utf-8') {
      try {
        return utf8Decoder.decode(bytes);
      } catch {
        return null;
      }
    }
    return null;
  }
};

const hasMostlyTextBytes = (bytes: Uint8Array) => {
  if (!bytes.length) return true;
  let controlCount = 0;
  for (const byte of bytes) {
    if (byte === 0) return false;
    if (byte < 0x20 && byte !== 0x09 && byte !== 0x0a && byte !== 0x0d) {
      controlCount += 1;
    }
  }
  return controlCount / bytes.length < 0.02;
};

export const isJSONContentType = (contentType: string | null) => {
  const normalized = normalizeContentType(contentType);
  return normalized.includes('application/json') || normalized.includes('+json');
};

export const isXMLContentType = (contentType: string | null) => {
  const normalized = normalizeContentType(contentType);
  return normalized.includes('application/xml') || normalized.includes('text/xml') || normalized.includes('+xml');
};

export const isHTMLContentType = (contentType: string | null) => {
  const normalized = normalizeContentType(contentType);
  return normalized.includes('text/html') || normalized.includes('application/xhtml+xml');
};

export const isImageContentType = (contentType: string | null) => baseContentType(contentType).startsWith('image/');

const isTextContentType = (contentType: string | null) => {
  const normalized = normalizeContentType(contentType);
  const mime = baseContentType(contentType);
  return mime.startsWith('text/')
    || mime === 'application/javascript'
    || mime === 'application/x-javascript'
    || mime === 'application/ecmascript'
    || mime === 'application/x-www-form-urlencoded'
    || mime === 'application/graphql-response+json'
    || mime === 'image/svg+xml'
    || isJSONContentType(normalized)
    || isXMLContentType(normalized);
};

const tryDecodeBase64Text = (body: string, charset: string = 'utf-8') => {
  const trimmed = body.trim();
  if (!trimmed) return null;
  try {
    const bytes = bytesFromBase64(trimmed);
    if (!hasMostlyTextBytes(bytes)) {
      return null;
    }
    return decodeTextBytes(bytes, charset);
  } catch {
    return null;
  }
};

// For known text content types, try base64 decode without the "mostly text" heuristic
const tryDecodeBase64ForTextContentType = async (body: string, contentEncoding: string | null, charset: string = 'utf-8') => {
  const trimmed = body.trim();
  if (!trimmed) return null;
  try {
    let bytes = bytesFromBase64(trimmed);
    if (contentEncoding) {
      bytes = await decompressBytes(bytes, contentEncoding);
    }
    // Skip hasMostlyTextBytes check - we already know this is a text content type
    return decodeTextBytes(bytes, charset);
  } catch {
    return null;
  }
};

const tryDecodeBase64TextAsync = async (body: string, contentEncoding: string | null, charset: string = 'utf-8') => {
  const trimmed = body.trim();
  if (!trimmed) return null;
  try {
    let bytes = bytesFromBase64(trimmed);
    if (contentEncoding) {
      bytes = await decompressBytes(bytes, contentEncoding);
    }
    if (!hasMostlyTextBytes(bytes)) {
      return null;
    }
    return decodeTextBytes(bytes, charset);
  } catch {
    return null;
  }
};

interface DecompressionStreamLike {
  readable: ReadableStream<Uint8Array>;
  writable: WritableStream<Uint8Array>;
}

type DecompressionStreamLikeConstructor = new (format: string) => DecompressionStreamLike;

const getDecompressionStreamConstructor = (): DecompressionStreamLikeConstructor | null => {
  const candidate = Reflect.get(globalThis, 'DecompressionStream');
  return typeof candidate === 'function' ? (candidate as DecompressionStreamLikeConstructor) : null;
};

const runDecompression = async (bytes: Uint8Array, format: string): Promise<Uint8Array> => {
  try {
    const DecompressionCtor = getDecompressionStreamConstructor();
    if (!DecompressionCtor) {
      return bytes;
    }

    const ds = new DecompressionCtor(format);
    const writer = ds.writable.getWriter();
    await writer.write(bytes);
    await writer.close();
    
    const reader = ds.readable.getReader();
    const chunks: Uint8Array[] = [];
    let totalLen = 0;
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      if (value) {
        chunks.push(value);
        totalLen += value.length;
      }
    }
    const result = new Uint8Array(totalLen);
    let offset = 0;
    for (const chunk of chunks) {
      result.set(chunk, offset);
      offset += chunk.length;
    }
    return result;
  } catch (e) {
    console.error(`Decompression failed for format ${format}`, e);
    return bytes;
  }
};

export const decompressBytes = async (bytes: Uint8Array, encoding: string): Promise<Uint8Array> => {
  const enc = encoding.toLowerCase().trim();
  if (enc === 'gzip' || enc === 'x-gzip') {
    return runDecompression(bytes, 'gzip');
  } else if (enc === 'deflate') {
    return runDecompression(bytes, 'deflate');
  } else if (enc === 'br') {
    try {
      if (getDecompressionStreamConstructor()) {
        try {
          return await runDecompression(bytes, 'br');
        } catch {
          // ignore
        }
      }
    } catch {}
  }
  return bytes;
};

export const decodeBodyAsync = async (body: string | null, contentType: string | null, contentEncoding: string | null = null) => {
  if (!body) return EMPTY_BODY_PLACEHOLDER;

  const trimmed = body.trim();
  const charset = extractCharset(contentType);

  if (isTextContentType(contentType)) {
    const decoded = await tryDecodeBase64ForTextContentType(trimmed, contentEncoding, charset);
    if (decoded) return decoded;
    return trimmed;
  }

  const decoded = await tryDecodeBase64TextAsync(trimmed, contentEncoding, charset);
  if (decoded) return decoded;

  return trimmed;
};

export const buildImageDataURLAsync = async (body: string | null, contentType: string | null, contentEncoding: string | null = null) => {
  if (!body || !isImageContentType(contentType)) {
    return null;
  }

  const mime = baseContentType(contentType) || 'image/*';
  if (mime === 'image/svg+xml') {
    const textBody = await decodeBodyAsync(body, contentType, contentEncoding);
    if (!textBody || textBody === EMPTY_BODY_PLACEHOLDER) {
      return null;
    }
    return `data:${mime};charset=utf-8,${encodeURIComponent(textBody)}`;
  }

  if (looksLikeBase64(body)) {
    return `data:${mime};base64,${body.trim()}`;
  }

  return `data:${mime};base64,${btoa(body)}`;
};

export const formatBodyForTextAsync = async (body: string | null, contentType: string | null, contentEncoding: string | null = null) => {
  const decodedBody = await decodeBodyAsync(body, contentType, contentEncoding);
  if (decodedBody === EMPTY_BODY_PLACEHOLDER) return decodedBody;

  if (isJSONContentType(contentType)) {
    try {
      return JSON.stringify(JSON.parse(decodedBody), null, 2);
    } catch {
      return decodedBody;
    }
  }

  return decodedBody;
};

const toPrimitiveNode = (value: unknown, label: string, id: string): BodyTreeNode => {
  if (value === null) {
    return { id, label, value: 'null', nodeType: 'null' };
  }

  const valueType = typeof value;
  if (valueType === 'string') {
    return { id, label, value: String(value), nodeType: 'string' };
  }
  if (valueType === 'number') {
    return { id, label, value: String(value), nodeType: 'number' };
  }
  if (valueType === 'boolean') {
    return { id, label, value: value ? 'true' : 'false', nodeType: 'boolean' };
  }

  return { id, label, value: String(value), nodeType: 'string' };
};

const buildJSONNode = (value: unknown, label: string, id: string): BodyTreeNode => {
  if (Array.isArray(value)) {
    return {
      id,
      label,
      nodeType: 'array',
      children: value.map((item, index) => buildJSONNode(item, `[${index}]`, `${id}.${index}`)),
    };
  }

  if (value && typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>);
    return {
      id,
      label,
      nodeType: 'object',
      children: entries.map(([key, child]) => buildJSONNode(child, key, `${id}.${key}`)),
    };
  }

  return toPrimitiveNode(value, label, id);
};

const parseJSONTree = (bodyText: string): StructuredBodyTree | null => {
  try {
    const parsed = JSON.parse(bodyText);
    return {
      kind: 'json',
      root: buildJSONNode(parsed, '$', '$'),
    };
  } catch {
    return null;
  }
};

const xmlNodeText = (node: ChildNode) => (node.textContent || '').trim();

const buildXMLNode = (element: Element, id: string): BodyTreeNode => {
  const attributes: BodyTreeNode[] = Array.from(element.attributes).map((attr) => ({
    id: `${id}.@${attr.name}`,
    label: `@${attr.name}`,
    value: attr.value,
    nodeType: 'xml-attribute',
  }));

  const childNodes: BodyTreeNode[] = [];
  let childElementIndex = 0;

  Array.from(element.childNodes).forEach((node, index) => {
    if (node.nodeType === Node.ELEMENT_NODE) {
      childElementIndex += 1;
      childNodes.push(buildXMLNode(node as Element, `${id}.${childElementIndex}`));
      return;
    }

    if (node.nodeType === Node.TEXT_NODE) {
      const textValue = xmlNodeText(node);
      if (textValue) {
        childNodes.push({
          id: `${id}.text.${index}`,
          label: '#text',
          value: textValue,
          nodeType: 'xml-text',
        });
      }
      return;
    }

    if (node.nodeType === Node.CDATA_SECTION_NODE) {
      const cdataValue = xmlNodeText(node);
      childNodes.push({
        id: `${id}.cdata.${index}`,
        label: '#cdata',
        value: cdataValue,
        nodeType: 'xml-cdata',
      });
      return;
    }

    if (node.nodeType === Node.COMMENT_NODE) {
      const commentValue = xmlNodeText(node);
      childNodes.push({
        id: `${id}.comment.${index}`,
        label: '#comment',
        value: commentValue,
        nodeType: 'xml-comment',
      });
    }
  });

  if (!attributes.length && !childNodes.length) {
    const value = (element.textContent || '').trim();
    return {
      id,
      label: element.tagName,
      value,
      nodeType: 'xml-element',
    };
  }

  return {
    id,
    label: element.tagName,
    nodeType: 'xml-element',
    children: [...attributes, ...childNodes],
  };
};

const parseXMLTree = (bodyText: string): StructuredBodyTree | null => {
  const parser = new DOMParser();
  const document = parser.parseFromString(bodyText, 'application/xml');
  if (document.querySelector('parsererror')) {
    return null;
  }

  const rootElement = document.documentElement;
  if (!rootElement) {
    return null;
  }

  return {
    kind: 'xml',
    root: buildXMLNode(rootElement, rootElement.tagName),
  };
};

export const parseStructuredBodyAsync = async (body: string | null, contentType: string | null, contentEncoding: string | null = null): Promise<StructuredBodyTree | null> => {
  const decodedBody = await decodeBodyAsync(body, contentType, contentEncoding);
  if (!decodedBody || decodedBody === EMPTY_BODY_PLACEHOLDER) {
    return null;
  }

  const trimmed = decodedBody.trim();
  if (!trimmed) {
    return null;
  }

  const jsonFirst = isJSONContentType(contentType) || (!isXMLContentType(contentType) && (trimmed.startsWith('{') || trimmed.startsWith('[')));
  const xmlFirst = isXMLContentType(contentType) || (!jsonFirst && trimmed.startsWith('<'));

  if (jsonFirst) {
    return parseJSONTree(trimmed) || parseXMLTree(trimmed);
  }

  if (xmlFirst) {
    return parseXMLTree(trimmed) || parseJSONTree(trimmed);
  }

  return parseJSONTree(trimmed) || parseXMLTree(trimmed);
};
