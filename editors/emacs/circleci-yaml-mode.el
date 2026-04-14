;;; circleci-yaml-mode.el --- Major mode for CircleCI YAML config files -*- lexical-binding: t; -*-

;; Copyright (C) 2025 CircleCI

;; Author: CircleCI
;; URL: https://github.com/CircleCI-Public/circleci-yaml-language-server
;; x-release-please-start-version
;; Version: 0.30.0
;; x-release-please-end
;; Package-Requires: ((emacs "28.1") (yaml-mode "0.0.13"))
;; Keywords: languages tools

;;; Commentary:

;; Major mode for editing CircleCI YAML configuration files, with
;; automatic language server integration.
;;
;; Derives from `yaml-mode' and activates for files under .circleci/
;; directories.  On first use, downloads the CircleCI YAML Language
;; Server binary and schema from GitHub releases.
;;
;; Works with both eglot (built-in from Emacs 29) and lsp-mode.
;; Whichever LSP client is loaded will be configured and started
;; automatically.
;;
;; Usage:
;;
;;   (require 'circleci-yaml-mode)
;;
;; Then open any .yml file under a .circleci/ directory.
;;
;; To enable private orb resolution, set your API token:
;;
;;   (setopt circleci-yaml-api-token "your-token-here")
;;
;; Or use dir-local variables / auth-source for per-project tokens.
;;
;; Known limitations:
;;
;; - Hover documentation is not provided by the language server.
;;   The VS Code extension implements hover via a separate JSON
;;   schema service.  A future version may add similar support.
;; - Self-hosted CircleCI Server URLs are not yet configurable.

;;; Code:

(require 'yaml-mode)
(require 'url)

(declare-function eglot-ensure "eglot")
(declare-function eglot-managed-p "eglot")
(declare-function eglot-current-server "eglot")
(declare-function jsonrpc-request "jsonrpc")
(declare-function lsp "lsp-mode")
(declare-function lsp-request "lsp-mode")
(declare-function lsp-register-client "lsp-mode")
(declare-function lsp-stdio-connection "lsp-mode")
(declare-function lsp-activate-on "lsp-mode")
(declare-function make-lsp-client "lsp-mode")

;;;; Customization

(defgroup circleci-yaml nil
  "CircleCI YAML configuration support."
  :group 'languages
  :prefix "circleci-yaml-")

;; x-release-please-start-version
(defcustom circleci-yaml-lsp-version "0.30.0"
  ;; x-release-please-end
  "Version of the CircleCI YAML Language Server to install."
  :type 'string)

(defcustom circleci-yaml-lsp-install-dir
  (expand-file-name "circleci-yaml-lsp" user-emacs-directory)
  "Directory where the language server binary and schema are installed."
  :type 'directory)

(defcustom circleci-yaml-api-token nil
  "CircleCI API token for private orb resolution and context support.
Get one from https://app.circleci.com/settings/user/tokens"
  :type '(choice (const :tag "None" nil)
                 string))

(defcustom circleci-yaml-start-lsp t
  "Whether to start the LSP client automatically.
When non-nil, the first available LSP client (eglot or lsp-mode)
is started when `circleci-yaml-mode' activates."
  :type 'boolean)

;;;; Platform detection

(defun circleci-yaml--platform ()
  "Return the platform string for GitHub release assets."
  (pcase system-type
    ('darwin "darwin")
    ('gnu/linux "linux")
    ('windows-nt "windows")
    (_ (error "circleci-yaml-mode: unsupported platform %s" system-type))))

(defun circleci-yaml--arch ()
  "Return the architecture string for GitHub release assets."
  (if (eq system-type 'windows-nt)
      "amd64"
    (let ((arch (string-trim (shell-command-to-string "uname -m"))))
      (pcase arch
        ((or "arm64" "aarch64") "arm64")
        ((or "x86_64" "amd64") "amd64")
        (_ (error "circleci-yaml-mode: unsupported architecture %s" arch))))))

(defun circleci-yaml--asset-name ()
  "Return the release asset filename for this platform."
  (let ((ext (if (eq system-type 'windows-nt) ".exe" "")))
    (format "%s-%s-lsp%s" (circleci-yaml--platform) (circleci-yaml--arch) ext)))

;;;; Paths

(defun circleci-yaml--lsp-bin ()
  "Return the path to the language server binary."
  (let ((ext (if (eq system-type 'windows-nt) ".exe" "")))
    (expand-file-name (concat "circleci-yaml-language-server" ext)
                      (expand-file-name "bin" circleci-yaml-lsp-install-dir))))

(defun circleci-yaml--schema-path ()
  "Return the path to schema.json."
  (expand-file-name "schema.json" circleci-yaml-lsp-install-dir))

(defun circleci-yaml--version-file ()
  "Return the path to the installed version marker file."
  (expand-file-name ".version" circleci-yaml-lsp-install-dir))

;;;; Installation

(defun circleci-yaml--installed-p ()
  "Return non-nil if the language server is installed."
  (and (file-executable-p (circleci-yaml--lsp-bin))
       (file-exists-p (circleci-yaml--schema-path))))

(defun circleci-yaml--installed-version ()
  "Return the installed version string, or nil."
  (let ((vf (circleci-yaml--version-file)))
    (when (file-exists-p vf)
      (string-trim (with-temp-buffer
                     (insert-file-contents vf)
                     (buffer-string))))))

(defun circleci-yaml--download-url (asset)
  "Return the GitHub release download URL for ASSET."
  (format "https://github.com/CircleCI-Public/circleci-yaml-language-server/releases/download/%s/%s"
          circleci-yaml-lsp-version asset))

(defun circleci-yaml--download (url dest)
  "Download URL to DEST, creating parent directories as needed."
  (let ((dir (file-name-directory dest)))
    (unless (file-exists-p dir)
      (make-directory dir t))
    (message "circleci-yaml-mode: downloading %s..." (file-name-nondirectory dest))
    (url-copy-file url dest t)))

(defun circleci-yaml-install ()
  "Download and install the CircleCI YAML Language Server."
  (interactive)
  (let ((bin (circleci-yaml--lsp-bin))
        (schema (circleci-yaml--schema-path))
        (version-file (circleci-yaml--version-file)))
    (circleci-yaml--download (circleci-yaml--download-url (circleci-yaml--asset-name)) bin)
    (set-file-modes bin #o755)
    (circleci-yaml--download (circleci-yaml--download-url "schema.json") schema)
    (with-temp-file version-file
      (insert circleci-yaml-lsp-version))
    (message "circleci-yaml-mode: installed v%s" circleci-yaml-lsp-version)))

(defun circleci-yaml-upgrade ()
  "Upgrade the language server if a newer version is configured.
Set `circleci-yaml-lsp-version' to the desired version before calling."
  (interactive)
  (let ((installed (circleci-yaml--installed-version)))
    (if (and installed (string= installed circleci-yaml-lsp-version))
        (message "circleci-yaml-mode: already at v%s" installed)
      (circleci-yaml-install))))

;;;; Derived mode

;;;###autoload
(define-derived-mode circleci-yaml-mode yaml-mode "CCI-YAML"
  "Major mode for editing CircleCI configuration files.
Derives from `yaml-mode'.  When an LSP client (eglot or lsp-mode)
is available, the CircleCI YAML Language Server is started
automatically, downloading it on first use if needed.")

;;;###autoload
(add-to-list 'auto-mode-alist '("/\\.circleci/.*\\.ya?ml\\'" . circleci-yaml-mode))

;;;; LSP client integration — eglot

(defun circleci-yaml--eglot-server-command ()
  "Return the eglot server command for the CircleCI YAML LS."
  (list (circleci-yaml--lsp-bin) "-stdio" "-schema" (circleci-yaml--schema-path)))

(with-eval-after-load 'eglot
  (add-to-list 'eglot-server-programs
               '(circleci-yaml-mode . circleci-yaml--eglot-contact))
  ;; Use a function contact so the paths are resolved at connection time,
  ;; after a potential auto-install.
  (defun circleci-yaml--eglot-contact (_interactive)
    "Return the eglot server contact for the CircleCI YAML LS."
    (circleci-yaml--eglot-server-command)))

;;;; LSP client integration — lsp-mode

(with-eval-after-load 'lsp-mode
  (require 'lsp-mode)
  (add-to-list 'lsp-language-id-configuration '(circleci-yaml-mode . "circleci-yaml"))
  (lsp-register-client
   (make-lsp-client
    :new-connection (lsp-stdio-connection #'circleci-yaml--eglot-server-command)
    :activation-fn (lsp-activate-on "circleci-yaml")
    :server-id 'circleci-yaml-ls)))

;;;; Token support

(defun circleci-yaml--maybe-send-token ()
  "Send the CircleCI API token to the language server if configured.
Only acts in `circleci-yaml-mode' buffers."
  (when (and (derived-mode-p 'circleci-yaml-mode)
             circleci-yaml-api-token)
    (cond
     ((and (fboundp 'eglot-managed-p) (eglot-managed-p))
      (require 'jsonrpc)
      (jsonrpc-request (eglot-current-server) :workspace/executeCommand
                       `(:command "setToken"
                         :arguments [,circleci-yaml-api-token])))
     ((and (fboundp 'lsp-request) (bound-and-true-p lsp-mode))
      (lsp-request "workspace/executeCommand"
                   `(:command "setToken"
                     :arguments [,circleci-yaml-api-token]))))))

(with-eval-after-load 'eglot
  (add-hook 'eglot-managed-mode-hook #'circleci-yaml--maybe-send-token))

(with-eval-after-load 'lsp-mode
  (add-hook 'lsp-after-initialize-hook #'circleci-yaml--maybe-send-token))

;;;; Auto-start

(defun circleci-yaml--ensure ()
  "Ensure the language server is installed and start the LSP client."
  (when circleci-yaml-start-lsp
    (unless (circleci-yaml--installed-p)
      (condition-case err
          (circleci-yaml-install)
        (error (message "circleci-yaml-mode: install failed: %s. Buffer will open without LSP."
                        (error-message-string err)))))
    (when (circleci-yaml--installed-p)
      (cond
       ((fboundp 'eglot-ensure) (eglot-ensure))
       ((fboundp 'lsp)          (lsp))
       (t (message "circleci-yaml-mode: no LSP client found (install eglot or lsp-mode)"))))))

(add-hook 'circleci-yaml-mode-hook #'circleci-yaml--ensure)

(provide 'circleci-yaml-mode)
;;; circleci-yaml-mode.el ends here
