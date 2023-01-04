import { configFileContent, configFileUri } from '../.env';

type VersionID = number;

class File {
  static #files: Record<string, File> = {};

  readonly content: string;

  readonly uri: string;

  /**
   * Latest version ID of the file generated.
   * This property is only handled by the original file.
   */
  #latestVersionId = 0;

  #versions : File[] = [];

  #original = this;

  #version:VersionID = 0;

  constructor(uri: string, content: string) {
    this.content = content;
    this.uri = uri;
  }

  get original() {
    return this.#original;
  }

  get version() {
    return this.#version;
  }

  getVersion(version: VersionID): File | undefined {
    if (!this.#isOriginal()) {
      return this.original.getVersion(version);
    }

    return this.#versions[version];
  }

  update(content: string) : File {
    if (!this.#isOriginal()) {
      return this.original.update(content);
    }

    const newFile = new File(this.uri, content);
    newFile.#version = this.#versions.length;
    newFile.#original = this.#original;

    this.#versions.push(newFile);

    return newFile;
  }

  #isOriginal(): boolean {
    return this.original === this;
  }

  #getNewVersionId() : number {
    if (!this.#isOriginal()) {
      return this.#original.#getNewVersionId();
    }

    this.#latestVersionId += 1;
    return this.#latestVersionId;
  }

  static async getFile(testingFile: string) {
    const uri = configFileUri(testingFile);

    let file = File.#files[uri];

    if (!file) {
      const content = await configFileContent(testingFile);
      file = new File(uri, content);
      File.#files[uri] = file;
    }

    return file;
  }
}

export default File;
