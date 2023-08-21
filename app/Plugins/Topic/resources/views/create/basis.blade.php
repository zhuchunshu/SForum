<div x-data="basis">
    <div class="mb-3">
        <label class="form-label">标题</label>
        <input type="text" x-model="title" id="topic-create-title" class="form-control"
               value="{{request()->input('basis.title')}}" name="basis[title]" placeholder="title" required>
    </div>

    <div class="mb-3">
        <label :class="{'required text-red' :! tag_selected}" class="form-label">选择板块</label>
        <select type="text" x-model="tag" name="basis[tag]" class="form-select"
                id="select-topic-tags"
                required>

            <option value="null"
                    data-custom-properties="&lt;span class=&quot;badge bg-danger-lt&quot;&gt;Tip&lt;/span&gt;">请选择板块
            </option>
            @foreach(\App\Plugins\Topic\src\Models\TopicTag::query()->where('status','=',null)->get() as $topic_tags)
                <option value="{{$topic_tags->id}}"
                        data-custom-properties="&lt;span class=&quot;badge&quot; style=&quot;background-color: {{$topic_tags->color}} &quot; &gt;{{$topic_tags->icon}}&lt;/span&gt;" @if((int)request()->input('basis.tag')===(int)$topic_tags->id)
                    {{"selected"}}
                        @endif>
                    {{$topic_tags->name}}
                </option>
            @endforeach
        </select>
    </div>


    <div class="mb-3">
        <label class="form-label">内容正文</label>
        <textarea name="basis[content]" id="basis-content" cols="30" rows="10"></textarea>
    </div>
    <div>
        @if(get_options('topic_emoji_close')!=='true')
            <link rel="stylesheet" href="{{file_hash('css/OwO.min.css')}}">
            <div class="OwO" id="create-comment-owo">[表情]</div>
            <script src="{{file_hash('js/editor.OwO.js')}}"></script>
        @endif
    </div>
</div>

<script src="{{file_hash("js/axios.min.js")}}"></script>
<script>
    function disableDefaultOption(select) {
        select.options[0].disabled = true; // 禁用默认选项
    }

    document.addEventListener('alpine:init', () => {
        Alpine.data('basis', () => ({
            tag_selected: false,
            init() {
                this.$watch('title', () => {
                    localStorage.setItem('create_topic_title', this.title)
                })
                this.$watch('tag', () => {
                    localStorage.setItem('create_topic_tag', this.tag)
                    this.tag_selected = this.tag !== 'null';
                })
                this.$nextTick(() => {
                    // 在 nextTick 中执行的代码
                    this.tag_selected = this.tag() !== 'null';
                });
            },
            title: (() => {
                if (localStorage.getItem('create_topic_title')) {
                    return localStorage.getItem('create_topic_title')
                }
                return null;
            }),
            tag: (() => {
                @if(!request()->has('basis.tag'))
                if (localStorage.getItem('create_topic_tag')) {
                    console.log(localStorage.getItem('create_topic_tag'))
                    if (localStorage.getItem('create_topic_tag') !== "null") {
                        this.tag_selected = true;
                    }
                    return localStorage.getItem('create_topic_tag')
                }
                @else
                return {{request()->input('basis.tag')}};
                @endif
            })
        }))
    })
</script>
<script defer>
    const target = document.getElementsByTagName("body")[0]
    const body_className = document.getElementsByTagName("body")[0].getAttribute("data-bs-theme");
    const observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            if (body_className !== document.getElementsByTagName("body")[0].getAttribute("data-bs-theme")) {
                setTimeout(()=>{
                    location.reload()
                },200)
            }
        });
    });

    observer.observe(target, {attributes: true});

    const image_upload_handler = (blobInfo, progress) => new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('file', blobInfo.blob(), blobInfo.filename());
        formData.append('_token', csrf_token);
        formData.append('_session', _token);
        axios.post("/user/upload/image", formData, {
            'Content-type': 'multipart/form-data'
        }).then(function (r) {
            console.log(r)
            const data = r.data;
            if (data.success) {
                resolve(data.result.url);
                return;
            }
            reject({message: 'HTTP Error: ' + data.result.msg + ', Error Code: ' + data.code, remove: true});
        }).catch(function (e) {
            console.log(e)
        })

    });

    document.addEventListener("DOMContentLoaded", function () {
        let options = {
            fullscreen_native: true,
            selector: '#basis-content',
            height: 500,
            menu: {!! \App\Plugins\Topic\src\Lib\Create\Editor::menu() !!},
            menubar: "{!! \App\Plugins\Topic\src\Lib\Create\Editor::menubar() !!}",
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
            convert_urls: false,
            setup: function (ed) {
                //console.log(ed)
            },
            external_plugins: {!! \App\Plugins\Topic\src\Lib\Create\Editor::externalPlugins() !!},
            images_upload_handler: image_upload_handler,
            init_instance_callback: (editor) => {
                if (localStorage.getItem('topic_create_content')) {
                    editor.setContent(localStorage.getItem('topic_create_content'))
                }
                editor.on('input', function (e) {
                    localStorage.setItem('topic_create_content', editor.getContent())
                });
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
            mobile: {
                menu: {!! \App\Plugins\Topic\src\Lib\Create\Editor::menu() !!},
                menubar: "{!! \App\Plugins\Topic\src\Lib\Create\Editor::menubar() !!}",
                toolbar_mode: 'scrolling',
                content_style: 'img{max-width:300px}'
            },
            branding: false,
            content_style: 'body { font-family: -apple-system, BlinkMacSystemFont, San Francisco, Segoe UI, Roboto, Helvetica Neue, sans-serif; font-size: 14px; -webkit-font-smoothing: antialiased; } img{max-width:700px}'
        }
        if (document.body.getAttribute("data-bs-theme") === 'dark') {
            options.skin = 'oxide-dark';
            options.content_css = 'dark';
        }
        tinyMCE.init(options);
    });
</script>