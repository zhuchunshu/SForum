<div class="mb-3">
    <label class="form-label">标题</label>
    <input type="text" class="form-control" value="{{request()->input('basis.title')}}" name="basis[title]" placeholder="title" required>
</div>

<div class="mb-3">
    <label class="form-label">选择标签</label>
    <select type="text" name="basis[tag]" class="form-select" id="select-topic-tags" required>
        <option value="null"
                data-custom-properties="">
            请选择
        </option>
        @foreach(\App\Plugins\Topic\src\Models\TopicTag::query()->where('status','=',null)->get() as $topic_tags)
            <option value="{{$topic_tags->id}}"
                    data-custom-properties="&lt;span class=&quot;badge&quot; style=&quot;background-color: {{$topic_tags->color}} &quot; &gt;{{$topic_tags->icon}}&lt;/span&gt;">
                {{$topic_tags->name}}
            </option>
        @endforeach
    </select>
</div>


<div class="mb-3">
    <label class="form-label">内容正文</label>
    <textarea name="basis[content]" id="basis-content" cols="30" rows="10"></textarea>
</div>

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


    document.addEventListener("DOMContentLoaded", function () {
        let options = {
            selector: '#basis-content',
            height: 450,
            menu:{!! \App\Plugins\Topic\src\Lib\Editor::menu() !!},
            menubar:"{!! \App\Plugins\Topic\src\Lib\Editor::menubar() !!}",
            statusbar: true,
            elementpath: true,
            promotion: false,
            plugins: {!! \App\Plugins\Topic\src\Lib\Editor::plugins() !!},
            language: "zh-Hans",
            toolbar: "{!! \App\Plugins\Topic\src\Lib\Editor::toolbar() !!}",
            link_default_target: '_blank',
            toolbar_mode: 'sliding',
            image_advtab: true,
            mobile:{
                menu:{!! \App\Plugins\Topic\src\Lib\Editor::menu() !!},
                menubar:"{!! \App\Plugins\Topic\src\Lib\Editor::menubar() !!}",
                toolbar_mode: 'scrolling'
            },
            autosave_ask_before_unload: true,
            autosave_interval: '1s',
            autosave_prefix: '{{config('codefec.app.name')}}-{path}{query}-{id}-',
            autosave_restore_when_empty: false,
            autosave_retention: '1400m',
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