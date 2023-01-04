import { commands } from './utils';

describe('Diagnostic testing files', () => {
  it('config1.yml', async () => {
    const testingFile = 'config1.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });

  it('autocomplete-jobs.yml', async () => {
    const testingFile = 'invalid-files/autocomplete-jobs.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });

  it('diagnostic.yml', async () => {
    const testingFile = 'diagnostic.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });
});
