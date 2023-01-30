/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/

tinymce.PluginManager.add('sfPreview', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '预览',
        size: "large",
        body: {
            type: 'panel',
            items: [
                {
                    type: 'iframe', // component type
                    name: 'sfPreview', // identifier
                    sandboxed: true,
                    transparent: true,
                }
            ]
        },
        buttons: [
            {
                type: 'cancel',
                text: 'Close'
            },
        ],
    });
    /* Add a button that opens a window */
    editor.ui.registry.addButton('sfPreview', {
        icon:'preview',
        onAction: () => {
            /* Open window */
            let _openDialog = openDialog();
            axios.post("/topic/create/comment/preview",{
                _token:csrf_token,
                content:editor.getContent()
            }).then(r=>{
                _openDialog.setData({
                    sfPreview:r.data
                })
            })
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sfPreview', {
        text: '预览',
        icon:'preview',
        onAction: () => {
            /* Open window */
            let _openDialog = openDialog();
            axios.post("/topic/create/comment/preview",{
                _token:csrf_token,
                content:editor.getContent()
            }).then(r=>{
                _openDialog.setData({
                    sfPreview:r.data
                })
            })
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
