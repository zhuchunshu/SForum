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
                @if(\Hyperf\Utils\Str::contains(core_http_url(),$data['parameter']))
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
            <li class="nav-item d-md-none ms-auto">
                <a href="/topic/create" class="btn btn-primary btn-pill shadow-sm py-1" role="button" rel="noreferrer">
                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-pencil" width="24"
                         height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                         stroke-linecap="round" stroke-linejoin="round">
                        <path d="M0 0h24v24H0z" stroke="none"/>
                        <path d="M4 20h4L18.5 9.5a1.5 1.5 0 0 0-4-4L4 16v4M13.5 6.5l4 4"/>
                    </svg>
                    发表</a>
            </li>
        </ul>
    </div>

    @if($page->count())
        @foreach($page as $data)
            @include('App::index.style2')
        @endforeach
    @else
        <div class="col-md-12">
            <div class="border-0 card card-body">
                <div class="text-center card-title">{{__("app.No more results")}}</div>
            </div>
        </div>
    @endif
    <div class="mt-2">
        {!! make_page($page) !!}
    </div>
</div>