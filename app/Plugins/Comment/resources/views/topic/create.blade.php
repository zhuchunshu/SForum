@extends('App::app')
@section('title','评论帖子【'.\Hyperf\Utils\Str::limit($topic->title,25).'】')
@section('content')
    <div class="row row-cards justify-content-center">
        <div class="col-lg-12">
            <div class="card">
                <div class="card-header">
                    <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                        <li class="breadcrumb-item"><a href="/">首页</a></li>
                        <li class="breadcrumb-item"><a href="/tags/{{$topic->tag->id}}.html">
                                {!! $topic->tag->icon !!}
                                {{$topic->tag->name}}
                            </a>
                        </li>
                        <li class="breadcrumb-item"><a href="/{{$topic->id}}.html">
                                {{\Hyperf\Utils\Str::limit($topic->title,25)}}
                            </a>
                        </li>
                        <li class="breadcrumb-item active" aria-current="page"><a href="#">回帖</a></li>
                    </ol>
                </div>
                <div class="card-header">
                    <h3 class="card-title">回帖</h3>
                </div>
                <div class="card-body">
                    <form action="/topic/create/comment/{{$topic->id}}" method="post">
                        <x-csrf/>
                        <input type="hidden" name="no_content" value="1">
                        <input type="hidden" name="topic_id" value="{{$topic->id}}">
                        <div class="mb-3">
                            <label for="" class="form-label"></label>
                            <textarea name="content" id="content" rows="3"></textarea>
                        </div>
                        <div class="mb-3 d-flex flex-row-reverse">
                            <button class="btn btn-primary" type="submit">回帖</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>
@endsection


@section('scripts')
    <script src="{{file_hash("js/axios.min.js")}}"></script>
    <script src="{{file_hash('tabler/libs/tinymce/tinymce.min.js')}}"></script>
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
                selector: '#content',
                height: 450,
                menu:{!! \App\Plugins\Comment\src\Lib\Create\Editor::menu() !!},
                menubar:"{!! \App\Plugins\Comment\src\Lib\Create\Editor::menubar() !!}",
                statusbar: true,
                elementpath: true,
                promotion: false,
                plugins: {!! \App\Plugins\Comment\src\Lib\Create\Editor::plugins() !!},
                language: "zh-Hans",
                toolbar: "{!! \App\Plugins\Comment\src\Lib\Create\Editor::toolbar() !!}",
                link_default_target: '_blank',
                toolbar_mode: 'sliding',
                image_advtab: true,
                automatic_uploads: true,
                convert_urls:false,
                external_plugins:{!! \App\Plugins\Comment\src\Lib\Create\Editor::externalPlugins() !!},
                images_upload_handler: image_upload_handler,
                init_instance_callback: (editor) => {
                    if(localStorage.getItem('create_topic_comment_{{$topic->id}}')){
                        editor.setContent(localStorage.getItem('create_topic_comment_{{$topic->id}}'))
                    }
                    editor.on('input', function(e) {
                        localStorage.setItem('create_topic_comment_{{$topic->id}}',editor.getContent())
                    });
                },
                mobile:{
                    menu:{!! \App\Plugins\Comment\src\Lib\Create\Editor::menu() !!},
                    menubar:"{!! \App\Plugins\Comment\src\Lib\Create\Editor::menubar() !!}",
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
@endsection