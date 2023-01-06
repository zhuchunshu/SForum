<div x-data="basis">
    <div class="mb-3">
        <label class="form-label">标题</label>
        <input type="text" x-model="title"  id="topic-create-title" class="form-control" value="{{request()->input('basis.title')}}" name="basis[title]" placeholder="title" required>
    </div>

    <div class="mb-3">
        <label class="form-label">选择标签</label>
        <select type="text" x-model="tag" name="basis[tag]" class="form-select" id="select-topic-tags" required>
            <option value="null"
                    data-custom-properties="">
                请选择
            </option>
            @foreach(\App\Plugins\Topic\src\Models\TopicTag::query()->where('status','=',null)->get() as $topic_tags)
                <option value="{{$topic_tags->id}}"
                        data-custom-properties="&lt;span class=&quot;badge&quot; style=&quot;background-color: {{$topic_tags->color}} &quot; &gt;{{$topic_tags->icon}}&lt;/span&gt;" @if(request()->input('basis.tag') && request()->input('basis.tag')==$topic_tags->id){{"selected"}}@endif>
                    {{$topic_tags->name}}
                </option>
            @endforeach
        </select>
    </div>


    <div class="mb-3">
        <label class="form-label">内容正文</label>
        <textarea name="basis[content]" id="basis-content" cols="30" rows="10"></textarea>
    </div>
</div>

<script src="{{file_hash("js/axios.min.js")}}"></script>
<script>
    document.addEventListener('alpine:init', () => {
        Alpine.data('basis', () => ({
            init() {
                this.$watch('title', () => {
                    localStorage.setItem('create_topic_title',this.title)
                })
                this.$watch('tag', () => {
                    localStorage.setItem('create_topic_tag',this.tag)
                })
            },
            title: (()=>{
                if(localStorage.getItem('create_topic_title')){
                    return localStorage.getItem('create_topic_title')
                }
                return null;
            }),
            tag:(()=>{
                if(localStorage.getItem('create_topic_tag')){
                    return localStorage.getItem('create_topic_tag')
                }
                return null;
            })
        }))
    })
</script>
<script defer>
    const target = document.getElementsByTagName("html")[0]
    const body_className = document.getElementsByTagName("html")[0].getAttribute("data-theme");
    const observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            if (body_className !== document.getElementsByTagName("html")[0].getAttribute("data-theme")) {
                location.reload()
            }
        });
    });

    observer.observe(target, {attributes: true});

    const image_upload_handler = (blobInfo, progress) => new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('file', blobInfo.blob(), blobInfo.filename());
        formData.append('_token', csrf_token);
        axios.post("/user/upload/image",formData,{
            'Content-type' : 'multipart/form-data'
        }).then(function(r){
            console.log(r)
            const data = r.data;
            if(data.success){
                resolve(data.result.url);
                return ;
            }
            reject({message:'HTTP Error: ' + data.result.msg + ', Error Code: '+data.code,remove: true});
        }).catch(function(e){
            console.log(e)
        })

    });

    document.addEventListener("DOMContentLoaded", function () {
        let options = {
            selector: '#basis-content',
            height: 450,
            menu:{!! \App\Plugins\Topic\src\Lib\Create\Editor::menu() !!},
            menubar:"{!! \App\Plugins\Topic\src\Lib\Create\Editor::menubar() !!}",
            statusbar: true,
            elementpath: true,
            promotion: false,
            plugins: {!! \App\Plugins\Topic\src\Lib\Create\Editor::plugins() !!},
            language: "zh-Hans",
            toolbar: "{!! \App\Plugins\Topic\src\Lib\Create\Editor::toolbar() !!}",
            link_default_target: '_blank',
            toolbar_mode: 'sliding',
            image_advtab: true,
            automatic_uploads: true,
            convert_urls:false,
            setup : function(ed) {
                //console.log(ed)
            },
            external_plugins:{!! \App\Plugins\Topic\src\Lib\Create\Editor::externalPlugins() !!},
            images_upload_handler: image_upload_handler,
            init_instance_callback: (editor) => {
                if(localStorage.getItem('topic_create_content')){
                    editor.setContent(localStorage.getItem('topic_create_content'))
                }
                editor.on('input', function(e) {
                    localStorage.setItem('topic_create_content',editor.getContent())
                });
            },
            mobile:{
                menu:{!! \App\Plugins\Topic\src\Lib\Create\Editor::menu() !!},
                menubar:"{!! \App\Plugins\Topic\src\Lib\Create\Editor::menubar() !!}",
                toolbar_mode: 'scrolling'
            },
            branding:false,
            content_style: 'body { font-family: -apple-system, BlinkMacSystemFont, San Francisco, Segoe UI, Roboto, Helvetica Neue, sans-serif; font-size: 14px; -webkit-font-smoothing: antialiased; }'
        }
        if (document.body.className === 'theme-dark') {
            options.skin = 'oxide-dark';
            options.content_css = 'dark';
        }
        tinyMCE.init(options);
    });
</script>