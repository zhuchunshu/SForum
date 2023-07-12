/*
  Note: We have included the plugin in the same JavaScript file as the TinyMCE
  instance for display purposes only. Tiny recommends not maintaining the plugin
  with the TinyMCE instance and using the `external_plugins` option.
*/

tinymce.PluginManager.add('sfVideo', (editor, url) => {
    const openDialog = () => editor.windowManager.open({
        title: '插入视频',
        size: "normal",
        body: {
            type: 'tabpanel',
            tabs: [ // array of tab panel configurations
                {
                    name: 'General',
                    title: 'General',
                    items: [
                        {
                            type: 'input', // component type
                            name: 'url', // identifier
                            inputMode: 'url',
                            label: '视频网址', // text for the label
                            placeholder: '请输入视频网址', // placeholder text for the input
                            maximized: false // grow width to take as much space as possible
                        },
                        {
                            type:"grid",
                            columns:2,
                            items:[
                                {
                                    type: 'input', // component type
                                    name: 'height', // identifier
                                    inputMode: 'number',
                                    label: '高度/px', // text for the label
                                    maximized: false // grow width to take as much space as possible
                                },
                                {
                                    type: 'input', // component type
                                    name: 'width', // identifier
                                    inputMode: 'number',
                                    value:"600",
                                    label: '宽度/px', // text for the label
                                    maximized: false // grow width to take as much space as possible
                                }
                            ]
                        }
                    ] // array of panel components
                },

                {
                    name: 'code',
                    title: 'code',
                    items: [
                        {
                            type:'textarea',
                            name: 'code',
                            label: '视频代码',
                            // enabled: false, // enabled state
                            maximized: true // grow width to take as much space as possible
                        }
                    ] // array of panel components
                },
                {
                    name: 'Embed',
                    title: 'Embed',
                    items: [
                        {
                            type: 'selectbox', // component type
                            name: 'embed', // identifier
                            label: '平台',
                            size: 1, // number of visible values (optional)
                            items: [
                                { value: 'bilibili', text: '哔哩哔哩' },
                                { value: 'youtube', text: 'YouTube' },
                            ]
                        },
                        {
                            type:'textarea',
                            name: 'embed_code',
                            label: '嵌入代码',
                            // enabled: false, // enabled state
                            maximized: true // grow width to take as much space as possible
                        }
                    ] // array of panel components
                },
            ]
        },
        buttons: [ // A list of footer buttons
            {
                type: 'cancel',
                name: 'closeButton',
                text: 'Cancel'
            },
            {
                type: 'submit',
                name: 'submitButton',
                text: 'insert',
                buttonType: 'primary'
            }
        ],
        onSubmit: (api) => {
            const data = api.getData();
            let url = data.url
            let code = data.code
            let embed = data.embed
            let embed_code = data.embed_code
            const width = data.width?data.width:600
            const height = data.height?data.height:400
            if (!url && !code && !(embed && embed_code)){
                swal({
                    "title":"至少输入一项",
                    "icon":"error"
                })
                return ;
            }
            // 如果url不为空
            if(url){
                // 判断url格式是否正确
                if(!isValidURL(url)){
                    swal({
                        "title":"网址格式不正确",
                        "icon":"error"
                    })
                    return ;
                }
                // 如果格式正确
                // 生成video代码
                let video = `<video src="${url}" width="${width}" height="${height}" controls="controls">您的浏览器不支持 video 标签。</video>`
                // 插入到编辑器
                editor.insertContent(video)
                api.close();
                return ;

            }
            // 如果code不为空
            if(code){
                // 判断代码格式是否正确，是否为有效的video格式
                if(!code.includes("<video")){
                    swal({
                        "title":"代码格式不正确",
                        "icon":"error"
                    })
                    return ;
                }
                // 判断video标签中是否有src属性
                if(!code.includes("src=")){
                    swal({
                        "title":"代码格式不正确，不包含src属性",
                        "icon":"error"
                    })
                    return ;
                }
                // 判断代码中是否包含width、height属性，如果没有则使用400和600
                if(!code.includes("width=")){
                    code = code.replace("<video","<video width="+width)
                }
                if(!code.includes("height=")){
                    code = code.replace("<video","<video height="+height)
                }
                // 插入到编辑器
                editor.insertContent(code)
                api.close();
                return ;
            }
            // 如果embed不为空
            if(embed){
                // 如果embed=bilibili
                if(embed === "bilibili"){
                    // 判断代码格式为<iframe src="//player.bilibili.com/player.html?aid=912834776&bvid=BV1GM4y177xe&cid=1178232875&page=1" scrolling="no" border="0" frameborder="no" framespacing="0" allowfullscreen="true"> </iframe>
                    if(!embed_code.includes("<iframe") || !embed_code.includes("src=") || !embed_code.includes("aid=") || !embed_code.includes("bilibili") || !embed_code.includes("bvid=")){
                        swal({
                            "title":"代码格式不正确",
                            "icon":"error"
                        })
                        return ;
                    }
                    // 获取src内容
                    let src = embed_code.split("src=")[1].split(" ")[0]
                    // 获取src中的aid
                    let aid = src.split("aid=")[1].split("&")[0]
                    // 生成bbcode代码
                    let bbcode = `[media website=bilibili]${aid}[/media]`
                    // 插入到编辑器
                    editor.insertContent(bbcode)
                    api.close();
                    return ;
                }
                // 如果embed=youtube
                if(embed === "youtube"){
                    // 判断代码格式为<iframe width="560" height="315" src="https://www.youtube.com/embed/5qap5aO4i9A" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
                    if(!embed_code.includes("<iframe") || !embed_code.includes("src=") || !embed_code.includes("youtube")){
                        swal({
                            "title":"代码格式不正确",
                            "icon":"error"
                        })
                        return ;
                    }
                    // 获取src内容
                    let src = embed_code.split("src=")[1].split(" ")[0]
                    // 获取"embed/"后面的内容
                    let id = src.split("embed/")[1]
                    // 如果id中包含'"'等符号则去掉
                    if(id.includes('"')){
                        id = id.split('"')[0]
                    }
                    // 生成bbcode代码
                    let bbcode = `[media website=youtube]${id}[/media]`
                    // 插入到编辑器
                    editor.insertContent(bbcode)
                    api.close();
                    return ;
                }
            }
            api.close();
        }
    });
    /* Add a button that opens a window */
    editor.ui.registry.addButton('sfVideo', {
        icon:'embed',
        onAction: () => {
            openDialog()
        }
    });
    /* Adds a menu item, which can then be included in any menu via the menu/menubar configuration */
    editor.ui.registry.addMenuItem('sfVideo', {
        text: '插入视频',
        icon:'embed',
        onAction: () => {
            /* Open window */
            openDialog()
        }
    });
    function isValidURL(string) {
        var res = string.match(/(http(s)?:\/\/.)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)/g);
        return (res !== null);
    }
    /* Return the metadata for the help plugin */
    return {
        getMetadata: () => ({
            name: 'sfVideo',
            url: 'https://www.runpod.cn'
        })
    };
});
