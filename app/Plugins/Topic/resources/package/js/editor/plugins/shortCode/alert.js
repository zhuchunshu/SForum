/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/
tinymce.PluginManager.add('sf-alert', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '警报提示',
        size: "large",
        body: {
            type: 'panel',
            items: [
                {
                    type: 'selectbox', // component type
                    name: 'type', // identifier
                    label: '类型',
                    size: 1, // number of visible values (optional)
                    items: [
                        { value: 'alert-success', text: 'success' },
                        { value: 'alert-info', text: 'info' },
                        { value: 'alert-warning', text: 'warning' },
                        { value: 'alert-error', text: 'error' },
                    ]
                },
                {
                    type: 'textarea',
                    name: 'content',
                    label: '内容'
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
            const alert_type = data.type;
            let content = data.content;
            content = content.replace(/\r?\n/g, "<br>");
            editor.insertContent('['+alert_type+']'+content+'[/'+alert_type+']');
            api.close();
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sf-alert', {
        text: '警报提示',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });
});
