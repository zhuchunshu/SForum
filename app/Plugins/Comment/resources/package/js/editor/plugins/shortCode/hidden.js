/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/
tinymce.PluginManager.add('sf-hidden', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '隐藏内容',
        size: "large",
        body: {
            type: 'panel',
            items: [
                {
                    type: 'selectbox', // component type
                    name: 'select', // identifier
                    label: '条件',
                    size: 1, // number of visible values (optional)
                    items: [
                        { value: 'login', text: '登陆可见' },
                        { value: 'reply', text: '回复可见' },
                        { value: 'only-author', text: '仅楼主可见' },
                        { value: 'password', text: '密码可见' },
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
            const select = data.select;
            if(select==="password"){
                localStorage.setItem('tinyMCE_plugin_sf_hidden_content', data.content);
                api.redial(openDialog2);
            }else{
                editor.insertContent('['+select+']'+data.content+'[/'+select+']');
                api.close();
            }
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sf-hidden', {
        text: '隐藏内容',
        onAction: () => {
            /* Open window */
            openDialog();
        }
    });


    const openDialog2 = {
        title: '输入查看密码',
        body: {
            type: 'panel',
            items: [
                {
                    type: 'input',
                    name: 'password',
                    label: '请输入查看密码'
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
            const content = localStorage.getItem('tinyMCE_plugin_sf_hidden_content');
            editor.insertContent('[password password='+data.password+']'+content+'[/password]');
            api.close();
        }
    };


    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sf-hidden', {
        text: '隐藏内容',
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