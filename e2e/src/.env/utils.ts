import fs from 'fs/promises';
import path from 'path';

const projectRoot = path.resolve(
  __dirname,
  '../../..',
);

const examplesFolder = path.resolve(
  projectRoot,
  'test-files',
);

function configFilePath(nameOrUri: string) {
  if (nameOrUri.startsWith('file://')) {
    return nameOrUri.substring(7);
  }

  return path.resolve(
    examplesFolder,
    '.circleci',
    nameOrUri,
  );
}

function configFileUri(name: string) {
  if (name.startsWith('file://')) {
    return name;
  }
  const filePath = configFilePath(name);
  return `file://${filePath}`;
}

function normalizePath(absolutePath: string): string {
  if (!absolutePath.startsWith('/')) {
    throw new Error('Not an absolute path');
  }

  const resolvedPath = path.resolve(absolutePath);

  if (!resolvedPath.startsWith(projectRoot)) {
    throw new Error(`Path ${resolvedPath} not in project root`);
  }

  return resolvedPath.replace(projectRoot, '/project');
}

function normalizeURI(uri: string): string {
  if (uri === '') {
    return '';
  }

  if (!uri.startsWith('file://')) {
    throw new Error('Not a valid URI');
  }

  const uriPath = uri.substring(5);

  return `file://${normalizePath(uriPath)}`;
}

async function configFileContent(nameOrUri: string, encoding: BufferEncoding = 'utf-8') {
  const filePath = configFilePath(nameOrUri);

  const buffer = await fs.readFile(filePath, encoding);

  return buffer.toString();
}

export {
  configFilePath,
  configFileContent,
  configFileUri,
  normalizePath,
  normalizeURI,
  projectRoot,
};
