@extends("app")
@section('title', $title)
@section('headerBtn')
    @foreach($headerBtn as $value)
        @include($value)
    @endforeach
@endsection
@section('scripts')
    @foreach($JsUrl as $value)
        <script src="{{ $value }}"></script>
    @endforeach
@endsection
@section('css')
    @foreach($CssUrl as $value)
        <link rel="stylesheet" href="{{ $value }}">
    @endforeach
@endsection
@section('content')
    {!! $body !!}
@endsection
