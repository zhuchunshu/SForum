/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/
tinymce.PluginManager.add('sf-code', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '插入代码',
        size: "large",
        body: {
            type: 'panel',
            items: [
                {
                    type: 'selectbox', // component type
                    name: 'lang', // identifier
                    label: 'Select Language',
                    size: 1, // number of visible values (optional)
                    items: [
                        { value: 'php', text: 'PHP' },
                        { value: 'javascript', text: 'JS' },
                        { value: 'java', text: 'Java' },
                        { value: 'python', text: 'Python' },
                        { value: 'kotlin', text: 'Kotlin' },
                        { value: 'html', text: 'html' },
                        { value: 'typescript', text: 'typescript' },
                        { value: 'css', text: 'css' },
                        { value: 'shell', text: 'shell' },
                        { value: 'bash', text: 'bash' },
                        { value: 'go', text: 'golang' },
                        { value: 'http', text: 'http' },
                        { value: 'cpp', text: 'cpp' },
                        { value: 'sql', text: 'sql' },
                        { value: 'swift', text: 'swift' },
                        { value: 'rust', text: 'rust' },
                        { value: 'ruby', text: 'ruby' },
                        { value: 'makefile', text: 'makefile' },
                        { value: 'ini', text: 'ini' },
                        { value: 'json', text: 'json' },
                    ]
                },
                {
                    type: 'textarea',
                    name: 'code',
                    label: 'Code'
                }
            ]
        },
        buttons: [
            {
                type: 'cancel',
                text: 'Close'
            },
            {
                type: 'submit',
                text: 'Insert',
                buttonType: 'primary'
            }
        ],
        onSubmit: (api) => {
            const data = api.getData();
            /* Insert content when the window form is submitted */
            editor.insertContent('[code lang='+data.lang+']'+data.code+'[/code]');
            api.close();
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sf-code', {
        text: '插入代码',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });
});

/*
  The following is an example of how to use the new plugin and the new
  toolbar button.
*/