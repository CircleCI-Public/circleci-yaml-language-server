import { Commands } from './types';
import {
  configFileUri,
  configFileContent,
  command,
  immediateDiagnostics,
} from './utils';

describe('DidOpen', () => {
  it('Job without steps', async () => {
    const response = await command(
      Commands.DocumentDidOpen,
      {
        textDocument: {
          text: await configFileContent('invalid-files/job-without-steps.yml'),
          uri: configFileUri('invalid-files/job-without-steps.yml'),
          version: 1,
          languageId: 'yaml',
        },
      },
    );

    expect(response).toBeNull();

    const diagnostics = await immediateDiagnostics();

    expect(diagnostics.list).toMatchSnapshot();
  });

  it('Unused job', async () => {
    const response = await command(
      Commands.DocumentDidOpen,
      {
        textDocument: {
          text: await configFileContent('invalid-files/unused-job.yml'),
          uri: configFileUri('invalid-files/unused-job.yml'),
          version: 1,
          languageId: 'yaml',
        },
      },
    );

    expect(response).toBeNull();

    const diagnostics = await immediateDiagnostics();

    expect(diagnostics).toMatchSnapshot();
  });

  it('Detects not existant dockers', async () => {
    const response = await command(
      Commands.DocumentDidOpen,
      {
        textDocument: {
          text: await configFileContent('invalid-files/bad-docker-image.yml'),
          uri: configFileUri('invalid-files/bad-docker-image.yml'),
          version: 1,
          languageId: 'yaml',
        },
      },
    );

    expect(response).toBeNull();

    const diagnostics = await immediateDiagnostics();

    expect(diagnostics).toMatchSnapshot();
  });
});
