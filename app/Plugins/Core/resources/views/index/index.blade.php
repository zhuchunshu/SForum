<div class="row row-cards justify-content-center">
    @if($page->count())
        @foreach($page as $data)
            <div class="col-md-10">
                <div class="border-0 card card-body">
                    <div class="row">
                        <div class="col-md-12">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="avatar" style="background-image: url({{super_avatar($data->user)}})"></span>
                                </div>
                                <div class="col">
                                    <div style="margin-bottom:0" class="card-title">{{$data->user->username}}</div>
                                    <div style="margin-top:1px">发布于:{{$data->created_at}}</div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-12">
                            <div class="row">
                                <div class="col-md-12 markdown">
                                    <a href="" class="text-reset"><h2>{{$data->title}}</h2></a>
                                    {{\Hyperf\Utils\Str::limit(core_default(deOptions($data->options)["summary"],"为捕获到本文摘要内容"),300)}}
                                    <div class="row">
                                        @foreach(deOptions($data->options)["images"] as $key=>$image)
                                            @if($key<=5)
                                                <div class="col-4">
                                                    <div class="border-5">
                                                        <a href="#" class="d-block"><img data-src="{{$image}}" class="card-img-top" alt="{{$image}}" src="{{get_options("topic_lazyload_img","/plugins/Topic/loading.gif")}}"></a>
                                                    </div>
                                                </div>
                                            @endif
                                        @endforeach
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-12" style="margin-top: 5px">
                            <div class="row">
                                <div class="col-auto bottomLine">
                                    <a href="/tags/{{$data->tag->id}}.html" style="text-decoration:none">
                                        <div class="card-circle">
                                            <img src="{{$data->tag->icon}}" alt="">
                                            <span>{{$data->tag->name}}</span>
                                        </div>
                                    </a>
                                </div>
                                <div class="col"></div>
                                <div class="col-auto" style="margin-top: 5px">
                                    <span>{{$data->view}} 浏览</span>
                                    <span>
                                        <button class="switch-icon switch-icon-flip" data-bs-toggle="switch-icon">
                              <span class="switch-icon-a text-muted">
                                <!-- Download SVG icon from http://tabler-icons.io/i/thumb-up -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="34" height="34" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M7 11v8a1 1 0 0 1 -1 1h-2a1 1 0 0 1 -1 -1v-7a1 1 0 0 1 1 -1h3a4 4 0 0 0 4 -4v-1a2 2 0 0 1 4 0v5h3a2 2 0 0 1 2 2l-1 5a2 3 0 0 1 -2 2h-7a3 3 0 0 1 -3 -3" /></svg>
                              </span>
                              <span class="switch-icon-b text-facebook">
                                <!-- Download SVG icon from http://tabler-icons.io/i/thumb-up -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="34" height="34" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M7 11v8a1 1 0 0 1 -1 1h-2a1 1 0 0 1 -1 -1v-7a1 1 0 0 1 1 -1h3a4 4 0 0 0 4 -4v-1a2 2 0 0 1 4 0v5h3a2 2 0 0 1 2 2l-1 5a2 3 0 0 1 -2 2h-7a3 3 0 0 1 -3 -3" /></svg>
                              </span>
                            </button>
                                        {{$data->like}}
                                    </span>
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
</div>