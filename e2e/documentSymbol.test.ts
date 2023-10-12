import { commands } from './utils';

describe('DocumentSymbol', () => {
  it('Standard file', async () => {
    const testingFile = 'config1.yml';
    await commands.didOpen(testingFile);

    const res = await commands.documentSymbol(testingFile);

    expect(res).toMatchSnapshot();
  });

  it('File with continuation workflow', async () => {
    const testingFile = 'continuation.yml';
    await commands.didOpen(testingFile);

    const res = await commands.documentSymbol(testingFile);

    expect(res).toMatchSnapshot();
  });
});
