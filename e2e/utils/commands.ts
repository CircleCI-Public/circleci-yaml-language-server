import * as utils from './helpers';
import {
  Commands,
} from './types';
import type {
  CompletionList,
  HoverCommandResponse,
  Position,
} from './types';
import DiagnosticList from './DiagnosticList';
import {
  configFileContent,
  configFileUri,
} from '../.env';

async function didOpen(
  filePath: string,
  version = 1,

  // Number of milliseconds to wait for orbs to be fetched
  waitOrbLoading = 3000,
) : Promise<DiagnosticList> {
  const response = await utils.command(
    Commands.DocumentDidOpen,
    {
      textDocument: {
        text: await configFileContent(filePath, 'utf-8'),
        uri: configFileUri(filePath),
        version,
        languageId: 'yaml',
      },
    },
  );

  const diagnostics = utils.immediateDiagnostics();

  expect(response).toBeNull();

  await new Promise((resolve) => {
    setTimeout(resolve, waitOrbLoading);
  });

  return diagnostics;
}

async function complete(
  filename: string,
  position: Position,
): Promise<CompletionList> {
  const response = await utils.command(
    Commands.Completion,
    {
      position,
      textDocument: {
        uri: configFileUri(filename),
      },
    },
  ) as CompletionList;

  response.items.sort((a, b) => a.label.localeCompare(b.label));

  return response;
}

async function hover(
  filename: string,
  position: Position,
): Promise<HoverCommandResponse> {
  const response = await utils.command(
    Commands.DocumentHover,
    {
      position,
      textDocument: {
        uri: configFileUri(filename),
      },
    },
  ) as HoverCommandResponse;

  return response;
}

export {
  complete,
  didOpen,
  hover,
};
