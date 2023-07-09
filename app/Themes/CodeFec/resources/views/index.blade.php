@extends("App::layouts.index-app")

@if($title)
    @section('title',$title." - ")
@endif


@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            @foreach(Itf()->get('ui-index-header-hook') as $k=>$v)
                @if(call_user_func($v['enable'])===true)
                    @include($v['view'])
                @endif
            @endforeach
            <div class="row row-cards justify-content-center">
                <div class="col-lg-9">
                    @foreach(Itf()->get('ui-index-content-hook') as $k=>$v)
                        @if(call_user_func($v['enable'])===true)
                            @include($v['view'])
                        @endif
                    @endforeach
                    @include('App::index.index')
                </div>
                <div class="col-lg-3 @if(get_options('theme_home_widget_hidden')==="true"){{"d-none d-lg-block"}}@endif">
                    <div class="row row-cards @if(get_options('theme_right_tool_sticky')!=='true'){{"rd"}}@endif">
                        <div class="col-md-12 @if(get_options('theme_right_tool_sticky')!=='true'){{"sticky"}}@endif" style="top: 105px">
                            @include('App::index.right')
                        </div>
                    </div>
                </div>
            </div>
        </div>


    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
{{--    <script src="/tabler/libs/apexcharts/dist/apexcharts.min.js"></script>--}}
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
