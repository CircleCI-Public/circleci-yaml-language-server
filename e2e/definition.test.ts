import {
  Commands,
  configFileUri,
  command,
  commands,
} from './utils';

describe('Definition', () => {
  it('Definition on parameter', async () => {
    const testingFile = 'config1.yml';
    const diagnostics = await commands.didOpen(testingFile);
    expect(diagnostics).toMatchSnapshot();

    const response = await command(Commands.Definition, {
      textDocument: {
        uri: configFileUri(testingFile),
      },
      position: {
        character: 16,
        line: 16,
      },
    });

    expect(response).toMatchSnapshot();
  });
});
