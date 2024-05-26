<div class="card">


    <div class="card-header">
        <ul class="nav nav-pills card-header-pills">
            @if(!count(request()->all()))
                <li class="nav-item">
                    <a class="nav-link active fw-bold" href="/">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-sm-block" width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path d="M0 0h24v24H0z" stroke="none"/>
                            <circle cx="12" cy="12" r="9"/>
                            <path d="M12 7v5l3 3"/>
                        </svg>
                        {{__('app.latest')}}</a>
                </li>
            @else
                <li class="nav-item">
                    <a class="nav-link" href="/">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-sm-block" width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path d="M0 0h24v24H0z" stroke="none"/>
                            <circle cx="12" cy="12" r="9"/>
                            <path d="M12 7v5l3 3"/>
                        </svg>
                        {{__('app.latest')}}</a>
                </li>
            @endif
            @foreach($topic_menu as $data)
                @if(\Hyperf\Stringable\Str::contains(core_http_url(),$data['parameter']))
                    <li class="nav-item">
                        <a class="nav-link active fw-bold" href="{{$data['url']}}">
                            {!!$data['icon']!!}{{$data['name']}}</a>
                    </li>
                @else
                    <li class="nav-item">
                        <a class="nav-link" href="{{$data['url']}}">
                            {!!$data['icon']!!}{{$data['name']}}</a>
                    </li>
                @endif
            @endforeach
            <li class="nav-item ms-auto">
                <div class="dropdown">
                    <a href="#" class="btn-action dropdown-toggle" data-bs-toggle="dropdown" aria-haspopup="true"
                       aria-expanded="false"><!-- Download SVG icon from http://tabler-icons.io/i/dots-vertical -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24"
                             stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                             stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                            <circle cx="12" cy="12" r="1"/>
                            <circle cx="12" cy="19" r="1"/>
                            <circle cx="12" cy="5" r="1"/>
                        </svg>
                    </a>
                    <div class="dropdown-menu dropdown-menu-end">
                        @foreach(Itf()->get('ui-home-tabs-dropdown') as $k=>$v)
                            @if(call_user_func($v['enable'])===true)
                                @include($v['view'])
                            @endif
                        @endforeach
                    </div>
                </div>
            </li>
        </ul>
    </div>

    @if($page->count())
        @foreach($page as $data)
            @include('App::index.style2')
        @endforeach
            <a href="/topic/create" class="border border-1 bg-primary  text-white position-fixed bottom-0 end-0 m-3 mb-6 mr-6 p-3 rounded-circle d-flex align-items-center justify-content-center d-lg-none" style="z-index: 9999">
                <svg xmlns="http://www.w3.org/2000/svg" width="35" height="35"  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon icon-tabler icons-tabler-outline icon-tabler-pencil-plus">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M4 20h4l10.5 -10.5a2.828 2.828 0 1 0 -4 -4l-10.5 10.5v4" />
                    <path d="M13.5 6.5l4 4" />
                    <path d="M16 19h6" />
                    <path d="M19 16v6" />
                </svg>
            </a>
        <div class="mt-2">
            {!! make_page($page) !!}
        </div>
    @else
        <div class="col-md-12">
            <div class="border-0 card card-body">
                <div class="text-center card-title">{{__("app.No more results")}}</div>
            </div>
        </div>
    @endif
</div>