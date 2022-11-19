import { Commands } from './types';
import {
  configFileUri,
  configFileContent,
  command,
  immediateDiagnostics,
} from './helpers';

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

    expect(diagnostics).toMatchSnapshot();
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
});
