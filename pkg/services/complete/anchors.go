package complete

func (ch *CompletionHandler) completeAnchors() {
	for name := range ch.Doc.YamlAnchors {
		ch.addCompletionItem(name)
	}
}
