@extends("App::app")

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
                <div class="col-md-9">
                    @foreach(Itf()->get('ui-index-content-hook') as $k=>$v)
                        @if(call_user_func($v['enable'])===true)
                            @include($v['view'])
                        @endif
                    @endforeach
                    @include('App::index.index')
                </div>
                <div class="col-md-3">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
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
    <script src="/tabler/libs/apexcharts/dist/apexcharts.min.js"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
