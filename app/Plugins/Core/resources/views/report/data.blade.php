@extends("Core::app")
@section('title','「'.$data->title.'」 的举报详情')
@section('content')
<div class="row justify-content-center">
    <div class="col-md-8">
        <div class="card border-0">
            <div class="card-body">
                <h3 class="card-title">{{$data->title}}</h3>
                <div class="markdown vditor-reset">
                    {!! $data->content !!}
                </div>
            </div>
            @if(auth()->check() && Authority()->check("admin_report"))
                <div class="card-footer" id="report-data-card-footer">
                    <button @click="submit" class="btn" :class="{'btn-primary':btn.class.isPrimary,'btn-danger':btn.class.isDanger}">@{{ btn.text }}</button>
                    <button style="margin-left:5px" @click="remove" class="btn btn-dark">删除</button>
                </div>
            @endif
        </div>
    </div>
</div>
@endsection

@section('scripts')
    <script> var report_id = "{{$data->id}}" </script>
    <script src="{{mix("plugins/Core/js/report.js")}}"></script>
@endsection