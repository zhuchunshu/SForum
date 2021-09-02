@extends("plugins.Core.app")

@section('title',"首页")


@section('content')

    <div class="row row-cards">
        <div class="col-md-9">
            @include('plugins.Core.index.index')
        </div>
        <div class="col-md-3">
            @include('plugins.Core.index.right')
        </div>
    </div>

@endsection
