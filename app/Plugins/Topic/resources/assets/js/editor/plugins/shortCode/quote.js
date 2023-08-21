/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/
tinymce.PluginManager.add('sf-quote', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '内容引用',
        body: {
            type: 'panel',
            items: [
                {
                    type: 'selectbox', // component type
                    name: 'type', // identifier
                    label: '类型',
                    size: 1, // number of visible values (optional)
                    items: [
                        { value: 'user', text: '用户' },
                        { value: 'topic', text: '帖子' },
                        { value: 'comment', text: '评论' },
                        { value: 'topic-tag', text: '板块' },
                    ]
                },
                {
                    type: 'input',
                    name: '_value',
                    label: '对应值 (用户id/帖子id/评论id等)'
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
            const type = data.type;
            const _value = data._value;
            if(type!=="topic-tag"){
                editor.insertContent('['+type+' '+type+'_id='+_value+'][/'+type+']');
            }else{
                editor.insertContent('['+type+' tag_id='+_value+'][/'+type+']');
            }
            api.close();
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sf-quote', {
        text: '内容引用',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });
});
