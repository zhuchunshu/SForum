@extends("app")

@section('title',"站点设置")

@section('content')
<div class="col-md-12">
    <div class="row row-cards" id="setting-core-form">
    <div class="col-md-12">
        <!-- Cards with tabs component -->
        <div class="card-tabs border-0">
            <!-- Cards navigation -->
            <ul class="nav nav-tabs">
                @foreach(Itf_Setting()->get() as $value)
                    <li class="nav-item"><a href="#{{$value['ename']}}" class="nav-link active" data-bs-toggle="tab">{{$value['name']}}</a></li>
                @endforeach
            </ul>
            <div class="tab-content" id="setting-core-form">
                <!-- Content of card #1 -->
                @foreach(Itf_Setting()->get() as $value)
                    @include($value['view'])
                @endforeach
            </div>
        </div>
    </div>
    <div class="col-md-12">
        <button @@click="submit" class="btn btn-primary">提交</button>
    </div>
    </div>
</div>
@endsection

@section('scripts')
    <script src="{{ mix("js/admin/setting.js") }}"></script>
@endsection