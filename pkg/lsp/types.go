package lsp

type DocumentURI string

type MarkupKind string

const (
	MarkupKindPlaintext MarkupKind = "plaintext"
	MarkupKindMarkdown  MarkupKind = "markdown"
)

type InitializeParams struct {
	ProcessID             int                `json:"process_id,omitempty"`
	RootPath              string             `json:"root_path,omitempty"`
	RootURI               DocumentURI        `json:"root_uri,omitempty"`
	InitializationOptions interface{}        `json:"initialization_options,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities,omitempty"`
	Trace                 bool               `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspace_folders,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync *TextDocumentSyncOptions `json:"text_document_sync,omitempty"`
	// HoverProvider is true if the server provides hover support.
	HoverProvider         bool                 `json:"hover_provider,omitempty"`
	CompletionProvider    *CompletionOptions   `json:"completion_provider,omitempty"`
	SignatureHelpProvider SignatureHelpOptions `json:"signature_help_provider,omitempty"`

	// DefinitionProvider is true if server provides goto defintion support
	DefinitionProvider             bool                         `json:"definition_provider,omitempty"`
	TypeDefintionProvider          bool                         `json:"type_defintion_provider,omitempty"`
	ImplementationProvider         bool                         `json:"implementation_provider,omitempty"`
	ReferencesProvider             bool                         `json:"references_provider,omitempty"`
	DocumentHighlightProvider      bool                         `json:"document_highlight_provider,omitempty"`
	DocumentSymbolProvider         bool                         `json:"document_symbol_provider,omitempty"`
	WorkspaceSymbolProvider        bool                         `json:"workspace_symbol_provider,omitempty"`
	CodeActionProvider             bool                         `json:"code_action_provider,omitempty"`
	CodeLensProvider               *CodeLensOptions             `json:"code_lens_provider,omitempty"`
	DocumentFormattingProvider     bool                         `json:"document_formatting_provider,omitempty"`
	DocumentRangeFormatterProvider bool                         `json:"document_range_formatter_provider,omitempty"`
	RenameProvider                 bool                         `json:"rename_provider,omitempty"`
	DocumentLinkProvider           *DocumentLinkOptions         `json:"document_link_provider,omitempty"`
	ColorProvider                  bool                         `json:"color_provider,omitempty"`
	FoldingRangeProvider           bool                         `json:"folding_range_provider,omitempty"`
	ExecuteCommandProvider         *ExecuteCommandOptions       `json:"execute_command_provider,omitempty"`
	Workspace                      *ServerCapabilitiesWorkspace `json:"workspace,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose         bool         `json:"open_close,omitempty"`
	Change            int          `json:"change,omitempty"`
	WillSave          bool         `json:"will_save,omitempty"`
	WillSaveWaitUntil bool         `json:"will_save_wait_until,omitempty"`
	Save              *SaveOptions `json:"save,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"include_text,omitempty"`
}

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolve_provider,omitempty"`
	TriggerCharacters []string `json:"trigger_characters,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"trigger_characters,omitempty"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolve_provider,omitempty"`
}

type DocumentLinkOptions struct {
	ResolveProvider bool `json:"resolve_provider,omitempty"`
}

type ExecuteCommandOptions struct {
	Commands []string `json:"commands,omitempty"`
}

type ServerCapabilitiesWorkspace struct {
	WorkspaceFolders *ServerCapabilitiesWorkspaceFolder `json:"workspace_folders,omitempty"`
}

type ServerCapabilitiesWorkspaceFolder struct {
	Supported          bool `json:"supported,omitempty"`
	ChangeNotification bool `json:"change_notification,omitempty"`
}

type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities   `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilites `json:"text_document,omitempty"`
	Experimental interface{}                    `json:"experimental,omitempty"`
}

type WorkspaceClientCapabilities struct {
	ApplyEdit              *bool                `json:"apply_edit,omitempty"`
	WorkspaceEdit          *DocumentChanges     `json:"workspace_edit,omitempty"`
	DidChangeConfiguration *DynamicRegistration `json:"did_change_configuration,omitempty"`
	DidChangeWatchedFiles  *DynamicRegistration `json:"did_change_watched_files,omitempty"`
	Symbol                 Symbol               `json:"symbol,omitempty"`
	ExecuteCommand         *DynamicRegistration `json:"execute_command,omitempty"`
	WorkspaceFolders       *bool                `json:"workspace_folders,omitempty"`
	Configuration          *bool                `json:"configuration,omitempty"`
}

type DocumentChanges struct {
	DocumentChanges *bool `json:"document_changes,omitempty"`
}

type DynamicRegistration struct {
	DynamicRegistration *bool `json:"dynamic_registration,omitempty"`
}

type Symbol struct {
	DynamicRegistration `json:"dynamic_registration,omitempty"`
	SymbolKind          SymbolKind `json:"symbol_kind,omitempty"`
}

type SymbolKind struct {
	ValueSet *SymbolKind `json:"value_set,omitempty"`
}

type TextDocumentClientCapabilites struct {
	Synchronization *Synchronization `json:"synchronization,omitempty"`
	Completion      *Completion      `json:"completion,omitempty"`
}

type Synchronization struct {
	*DynamicRegistration
	WillSave          *bool `json:"will_save,omitempty"`
	WillSaveWaitUntil *bool `json:"will_save_wait_until,omitempty"`
	DidSave           *bool `json:"did_save,omitempty"`
}

type Completion struct {
	*DynamicRegistration
	CompletionItem     *CompletionItem      `json:"completion_item,omitempty"`
	CompletionItemKind *CompletionItemKind  `json:"completion_item_kind,omitempty"`
	Hover              *Hover               `json:"hover,omitempty"`
	SignatureHelp      *SignatureHelp       `json:"signature_help,omitempty"`
	References         *DynamicRegistration `json:"references,omitempty"`
	DocumentHighlight  *DynamicRegistration `json:"document_highlight,omitempty"`
	DocumenetSymbol    *DocumentSymbol      `json:"documenet_symbol,omitempty"`
	Formatting         *DynamicRegistration `json:"formatting,omitempty"`
	RangeFormatting    *DynamicRegistration `json:"range_formatting,omitempty"`
	OnTypeFormatting   *DynamicRegistration `json:"on_type_formatting,omitempty"`
	Definition         *DynamicRegistration `json:"definition,omitempty"`
	TypeDefinition     *DynamicRegistration `json:"type_definition,omitempty"`
	Implementation     *DynamicRegistration `json:"implementation,omitempty"`
	CodeAction         *CodeAction          `json:"code_action,omitempty"`
	CodeLens           *DynamicRegistration `json:"code_lens,omitempty"`
	DocumentLink       *DynamicRegistration `json:"document_link,omitempty"`
	ColorProvider      *DynamicRegistration `json:"color_provider,omitempty"`
	Rename             *DynamicRegistration `json:"rename,omitempty"`
	PublishDiagnostics *PublishDiagnostics  `json:"publish_diagnostics,omitempty"`
}

type CompletionItem struct {
	SnippetSupport          *bool        `json:"snippet_support,omitempty"`
	CommitCharactersSupport *bool        `json:"commit_characters_support,omitempty"`
	DocumentationFormat     []MarkupKind `json:"documentation_format,omitempty"`
	DeprecatedSupport       *bool        `json:"deprecated_support,omitempty"`
	PreselectSupport        *bool        `json:"preselect_support,omitempty"`
}

type CompletionItemKind struct {
	ValueSet []CompletionItemKind `json:"value_set,omitempty"`
}

type Hover struct {
	*DynamicRegistration
	ContentFormat []MarkupKind `json:"content_format,omitempty"`
}

type SignatureHelp struct {
	*DynamicRegistration
	SignatureInformation *SignatureInformation `json:"signature_information,omitempty"`
}

type SignatureInformation struct {
	DocumentationFormat []MarkupKind `json:"documentation_format,omitempty"`
}

type DocumentSymbol struct {
	*DynamicRegistration
	SymbolKind                        *SymbolKind `json:"symbol_kind,omitempty"`
	HierarchicalDocumentSymbolSupport *bool       `json:"hierarchical_document_symbol_support,omitempty"`
}

type CodeAction struct {
	*DynamicRegistration
	CodeActionLiteralSupport *CodeActionLiteralSupport `json:"code_action_literal_support,omitempty"`
}

type CodeActionLiteralSupport struct {
	CodeActionKind CodeActionKind `json:"code_action_kind,omitempty"`
}

type CodeActionKind struct {
	ValueSet []CodeActionKind `json:"value_set,omitempty"`
}

type PublishDiagnostics struct {
	RelatedInformation *bool `json:"related_information,omitempty"`
}

type FoldingRange struct {
	*DynamicRegistration
	RangeLimit      *int  `json:"range_limit,omitempty"`
	LineFoldingOnly *bool `json:"line_folding_only,omitempty"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri,omitempty"`
	Name string `json:"name,omitempty"`
}
