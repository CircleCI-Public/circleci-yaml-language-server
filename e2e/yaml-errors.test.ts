import {
  commands,
} from './utils';

describe('YAML Errors', () => {
  it('parameter key not defined', async () => {
    const testingFile = 'invalid-files/yaml-errors.yml';
    const diagnostics = await commands.didOpen(testingFile);

    diagnostics.includes({
      message: 'Did not find expected key',
      range: {
        start: { line: 3, character: 2 },
        end: { line: 3, character: 5 },
      },
    });

    expect(diagnostics).toMatchSnapshot();
  });
});
