/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/

tinymce.PluginManager.add('sfEmoji', (editor, url) => {
    const tabs = ()=>{
        return {
            name:"a",
            title:"Title",
            items: [
                {
                    type: 'iframe', // component type
                    name: 'sfPreview', // identifier
                    sandboxed: true,
                    transparent: true,
                }
            ]
        }
    }
    const openDialog = () => editor.windowManager.open({
        title: '表情',
        size: "large",
        body: {
            type: 'tabpanel',
            tabs:[
                tabs()
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
    editor.ui.registry.addButton('sfEmoji', {
        icon:'emoji',
        onAction: () => {
            /* Open window */
            let _openDialog = openDialog();
            axios.post("/topic/create/preview",{
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
    editor.ui.registry.addMenuItem('sfEmoji', {
        text: '表情',
        icon:'emoji',
        onAction: () => {
            /* Open window */
            let _openDialog = openDialog();
            axios.post("/topic/create/preview",{
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
            name: 'sfEmoji',
            url: 'https://www.runpod.cn'
        })
    };
});
