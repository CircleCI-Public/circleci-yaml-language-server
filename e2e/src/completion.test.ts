import { Commands } from './types';
import {
  configFileUri,
  command,
} from './helpers';

describe('Completion command', () => {
  it('Completion command', async () => {
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
