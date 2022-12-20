/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/
tinymce.PluginManager.add('sfPreview', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: 'Preview',
        type: 'htmlpanel', // component type
        html: '<div>Html goes here</div>',
        body: {
            type: 'panel',
            items: [
                {
                    type: 'input',
                    name: 'title',
                    label: 'Title'
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
                text: 'Save',
                buttonType: 'primary'
            }
        ],
        onSubmit: (api) => {
            const data = api.getData();
            /* Insert content when the window form is submitted */
            editor.insertContent('Title: ' + data.title);
            api.close();
        }
    });
    /* Add a button that opens a window */
    editor.ui.registry.addButton('sfPreview', {
        icon:'preview',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sfPreview', {
        text: '预览',
        icon:'preview',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });
    /* Return the metadata for the help plugin */
    return {
        getMetadata: () => ({
            name: 'sfPreview',
            url: 'https://www.runpod.cn'
        })
    };
});

/*
  The following is an example of how to use the new plugin and the new
  toolbar button.
*/
tinymce.init({
    selector: 'textarea#custom-plugin',
    plugins: 'example help',
    toolbar: 'example | help'
});