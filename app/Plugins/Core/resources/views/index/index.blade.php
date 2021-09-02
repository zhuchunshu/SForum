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
                                            @if($key<=8)
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
                        <div class="col-md-12">
                            <div class="row">

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