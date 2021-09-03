@extends("plugins.Core.app")

@section('title',"首页")


@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('plugins.Core.index.index')
                </div>
                <div class="col-md-5">
                    @include('plugins.Core.index.right')
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
