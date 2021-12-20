<div class="row row-cards justify-content-center">
    @if($page->count())
        @foreach($page as $data)
            <div class="col-md-12">
                <div class="border-0 card card-body">
                    <div class="row">
                        <div class="col-md-12">
                            <div class="row">
                                <div class="col-auto">
                                    <a href="/users/{{$data->user->username}}.html" class="avatar" style="background-image: url({{super_avatar($data->user)}})"></a>
                                </div>
                                <div class="col">
                                    <a href="/users/{{$data->user->username}}.html" style="margin-bottom:0;text-decoration:none;" class="card-title text-reset">{{$data->user->username}}</a>
                                    <div style="margin-top:1px">发布于:{{$data->created_at}}</div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-12">
                            <div class="row">
                                <div class="col-md-12 markdown home-article">
                                    <a href="/{{$data->id}}.html" class="text-reset"><h2>{{$data->title}}</h2></a>
                                    <span class="home-summary">{{\Hyperf\Utils\Str::limit(core_default(deOptions($data->options)["summary"],"未捕获到本文摘要"),300)}}</span>
                                    <div class="row">
                                        @foreach(deOptions($data->options)["images"] as $key=>$image)
                                            @if($key<=5)
                                                <div class="col-4">
                                                    <div class="border-5">
                                                        <a href="/{{$data->id}}.html" class="d-block"><img data-src="{{$image}}" class="card-img-top" alt="{{$image}}" src="{{get_options("topic_lazyload_img","/plugins/Topic/loading.gif")}}"></a>
                                                    </div>
                                                </div>
                                            @endif
                                        @endforeach
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-12" style="margin-top: 5px">
                            <div class="d-flex align-items-center">
                                <div class="col-auto bottomLine">
                                    <a href="/tags/{{$data->tag->id}}.html" style="text-decoration:none">
                                        <div class="card-circle">
                                            <img src="{{$data->tag->icon}}" alt="">
                                            <span>{{$data->tag->name}}</span>
                                        </div>
                                    </a>
                                </div>
                                <div class="ms-auto">
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="浏览量">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><circle cx="12" cy="12" r="2" /><path d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7" /></svg>
                                        {{$data->view}}
                                    </span>
                                    <a href="#" class="ms-3 text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="点赞">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" /></svg>
                                        {{$data->like}}
                                    </a>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        @endforeach
    @else
        <div class="col-md-10">
            <div class="border-0 card card-body">
                <div class="text-center card-title">暂无内容</div>
            </div>
        </div>
    @endif
    {!! make_page($page) !!}
</div>