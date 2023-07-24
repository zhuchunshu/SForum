<div class="mb-3">
    <label class="form-label">标题</label>
    <input type="text" class="form-control"  value="{{$data->title}}" name="basis[title]" placeholder="title" required>
</div>

<input type="hidden" name="basis[topic_id]" value="{{$data->id}}">
<div class="mb-3">
    <label class="form-label">选择标签</label>
    <select type="text" name="basis[tag]" class="form-select" id="select-topic-tags" required>
        @foreach(\App\Plugins\Topic\src\Models\TopicTag::query()->where('status','=',null)->get() as $topic_tags)
            <option value="{{$topic_tags->id}}"
                    data-custom-properties="&lt;span class=&quot;badge&quot; style=&quot;background-color: {{$topic_tags->color}} &quot; &gt;{{$topic_tags->icon}}&lt;/span&gt;" @if($data->tag->id===$topic_tags->id){{"selected"}}@endif>
                {{$topic_tags->name}}
            </option>
        @endforeach
    </select>
</div>


<div class="mb-3">
    <label class="form-label">内容正文</label>
    <textarea name="basis[content]" id="basis-content" cols="30" rows="10">{{$data->post->content}}</textarea>
</div>
<div>
    @if(get_options('topic_emoji_close')!=='true')
        <link rel="stylesheet" href="{{file_hash('css/OwO.min.css')}}">
        <div class="OwO" id="create-comment-owo">[表情]</div>
        <script src="{{file_hash('js/editor.OwO.js')}}"></script>
    @endif
</div>

<script src="{{file_hash("js/axios.min.js")}}"></script>
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
        formData.append('_session', _token);
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
            fullscreen_native: true,
            selector: '#basis-content',
            height: 500,
            menu:{!! \App\Plugins\Topic\src\Lib\Edit\Editor::menu() !!},
            menubar:"{!! \App\Plugins\Topic\src\Lib\Edit\Editor::menubar() !!}",
            statusbar: true,
            elementpath: true,
            promotion: false,
            plugins: {!! \App\Plugins\Topic\src\Lib\Edit\Editor::plugins() !!},
            language: "zh-Hans",
            toolbar: "{!! \App\Plugins\Topic\src\Lib\Edit\Editor::toolbar() !!}",
            link_default_target: '_blank',
            toolbar_mode: 'sliding',
            image_advtab: true,
            automatic_uploads: true,
            convert_urls:false,
            setup : function(ed) {
                //console.log(ed)
            },
            external_plugins:{!! \App\Plugins\Topic\src\Lib\Edit\Editor::externalPlugins() !!},
            images_upload_handler: image_upload_handler,
            init_instance_callback:(editor)=>{
                @if(get_options('topic_emoji_close')!=='true')
                new OwO({
                    logo: '[OωO表情]',
                    container: document.getElementById('create-comment-owo'),
                    target: editor,
                    api: '/api/core/OwO.json',
                    width: '300px',
                    maxHeight: '250px',
                });
                @endif
            },
            mobile:{
                menu:{!! \App\Plugins\Topic\src\Lib\Edit\Editor::menu() !!},
                menubar:"{!! \App\Plugins\Topic\src\Lib\Edit\Editor::menubar() !!}",
                toolbar_mode: 'scrolling',
                content_style: 'img{max-width:300px}'
            },
            autosave_ask_before_unload: true,
            autosave_interval: '1s',
            autosave_prefix: '{{config('codefec.app.name')}}-{path}{query}-{id}-',
            autosave_restore_when_empty: false,
            autosave_retention: '1400m',
            branding:false,
            content_style: 'body { font-family: -apple-system, BlinkMacSystemFont, San Francisco, Segoe UI, Roboto, Helvetica Neue, sans-serif; font-size: 14px; -webkit-font-smoothing: antialiased; } img{max-width:700px}'
        }
        if (document.body.getAttribute("data-bs-theme") === 'dark') {
            options.skin = 'oxide-dark';
            options.content_css = 'dark';
        }
        tinyMCE.init(options);
    });
</script>