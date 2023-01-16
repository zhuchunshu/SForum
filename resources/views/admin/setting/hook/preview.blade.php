@extends('App::app')
@section('title','小部件预览')
@section('content')
<div class="row row-cards">
    <div class="card">
        <div class="card-header">
            <h3 class="card-title">小部件预览</h3>
        </div>
        <div class="card-body">
            @include($view)
        </div>
    </div>
</div>
@endsection