@extends("App::app")

@section('title',"修改评论")


@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-lg-12">
            <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                <li class="breadcrumb-item"><a href="/">首页</a></li>
                <li class="breadcrumb-item"><a href="/tags/{{$comment->topic->tag->id}}.html">
                        {!! $comment->topic->tag->icon !!}
                        {{$comment->topic->tag->name}}
                    </a>
                </li>
                <li class="breadcrumb-item"><a href="/{{$comment->topic->id}}.html">
                        {{\Hyperf\Utils\Str::limit($comment->topic->title,25)}}
                    </a>
                </li>
                <li class="breadcrumb-item"><a href="{{'/' . $comment->topic_id . '.html/' . $comment->id . '?page=' . get_topic_comment_page($comment->id)}}">
                        ID【{{$comment->id}}】的评论
                    </a>
                </li>
                <li class="breadcrumb-item active" aria-current="page"><a href="#">修改评论</a></li>
            </ol>
        </div>
        <div class="col-lg-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">回帖</h3>
                </div>
                <div class="card-body">
                    <form action="/comment/topic/{{$comment->id}}/edit" method="post">
                        <x-csrf/>
                        <input type="hidden" name="comment_id" value="{{$comment->id}}">
                        <div class="mb-3">
                            <label for="" class="form-label"></label>
                            <textarea name="content" id="content" rows="3">{!! $comment->post->content !!}</textarea>
                        </div>
                        <div class="mb-3 d-flex flex-row-reverse">
                            <button class="btn btn-primary" type="submit">修改评论</button>
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
                menu:{!! \App\Plugins\Comment\src\Lib\Edit\Editor::menu() !!},
                menubar:"{!! \App\Plugins\Comment\src\Lib\Edit\Editor::menubar() !!}",
                statusbar: true,
                elementpath: true,
                promotion: false,
                plugins: {!! \App\Plugins\Comment\src\Lib\Edit\Editor::plugins() !!},
                language: "zh-Hans",
                toolbar: "{!! \App\Plugins\Comment\src\Lib\Edit\Editor::toolbar() !!}",
                link_default_target: '_blank',
                toolbar_mode: 'sliding',
                image_advtab: true,
                automatic_uploads: true,
                convert_urls:false,
                external_plugins:{!! \App\Plugins\Comment\src\Lib\Edit\Editor::externalPlugins() !!},
                images_upload_handler: image_upload_handler,
                mobile:{
                    menu:{!! \App\Plugins\Comment\src\Lib\Edit\Editor::menu() !!},
                    menubar:"{!! \App\Plugins\Comment\src\Lib\Edit\Editor::menubar() !!}",
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