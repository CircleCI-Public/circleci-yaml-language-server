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

  it('Matrix alias', async () => {
    const testingFile = 'matrix-alias.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });

  it('Inline parameter', async () => {
    const testingFile = 'parameter-inline.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });

  it('Undefined parameter', async () => {
    const testingFile = 'invalid-files/not-defined-param.yml';
    const diagnostics = await commands.didOpen(testingFile);

    expect(diagnostics).toMatchSnapshot();
  });
});
