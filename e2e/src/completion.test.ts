import { Commands } from './types';
import {
  configFileUri,
  command,
  didOpen,
} from './helpers';

describe('Completion command', () => {
  it('Completion command', async () => {
    // Opening the file before working on it
    await didOpen('config1.yml');

    const response = await command(
      Commands.Completion,
      {
        position: {
          character: 1,
          line: 1,
        },
        textDocument: {
          uri: configFileUri('config1.yml'),
        },
      },
    );

    expect(response).toMatchSnapshot();
  });
});
