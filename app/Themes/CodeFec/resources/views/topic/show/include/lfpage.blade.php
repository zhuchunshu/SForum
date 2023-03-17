@if(get_options('topic_previous_page_close')!=='true')
<script>
    document.addEventListener('alpine:init', () => {
        Alpine.data('topic_show_include_ifpage', () => ({
            data: {'shang':null,'xia':null},
            async init() {
                this.data = await (await fetch('/api/topic/get.topic.include.ifpage', {
                    'method': 'post', headers: {
                        'Content-Type': 'application/json'
                    }, body: JSON.stringify({_token: csrf_token, topic_id: '{{$data->id}}'})
                })).json().then(res => res.result.result);
            },
        }))
    })
</script>
<div class="col-md-12">
    <div class="border-0 card">
        <div class="card-body">
            <ul class="pagination " x-data="topic_show_include_ifpage">
                <template x-if="data.shang && data.shang.id">
                    <li class="page-item page-prev">
                        <a class="page-link"
                           :href="data.shang.url">
                            <div class="page-item-subtitle">{{__("topic.previous")}}</div>
                            <div class="page-item-title text-reset" x-text="data.shang.title"></div>
                        </a>
                    </li>
                </template>
                <template x-if="data.shang===null || !data.shang.id">
                    <li class="page-item page-prev disabled">
                        <a class="page-link" href="#" tabindex="-1" aria-disabled="true">
                            <div class="page-item-subtitle">{{__("topic.previous")}}</div>
                            <div class="page-item-title">{{__("app.none")}}</div>
                        </a>
                    </li>
                </template>

                <template x-if="data.xia && data.xia.id">
                    <li class="page-item page-next">
                        <a class="page-link"
                           :href="data.xia.url">
                            <div class="page-item-subtitle">{{__("topic.next")}}</div>
                            <div class="page-item-title text-reset" x-text="data.xia.title"></div>
                        </a>
                    </li>
                </template>
                <template x-if="data.xia===null || !data.xia.id">
                    <li class="page-item page-next disabled">
                        <a class="page-link" href="#">
                            <div class="page-item-subtitle">{{__("topic.next")}}</div>
                            <div class="page-item-title">{{__("app.none")}}</div>
                        </a>
                    </li>
                </template>
            </ul>
        </div>
    </div>
</div>
@endif