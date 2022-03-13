@extends("Core::app")

@if($title)
    @section('title',$title." - ")
@endif


@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('Core::index.index')
                </div>
                <div class="col-md-5">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('Core::index.right')
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <div>
            <a href="/topic/create" class="btn btn-floating btn-indigo" style="opacity: 0.81;bottom: 45px; right: 24px;-moz-border-radius: 50px;-webkit-border-radius: 50px;border-radius: 50px;">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <line x1="12" y1="5" x2="12" y2="19"></line>
                    <line x1="5" y1="12" x2="19" y2="12"></line>
                </svg>
                发帖
            </a>
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
