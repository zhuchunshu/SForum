<div class="container-fluid">
    <!-- Page title -->
    <div class="page-header d-print-none">
        @foreach(Itf()->get('admin-ui-header-title-hook') as $key=>$value)
            @if(call_user_func($value['enable'])===true)
                <div id="{{$key}}">
                    @include($value['view'])
                </div>
            @endif
        @endforeach
        <div class="row align-items-center">
            <div class="col">
                <!-- Page pre-title -->
                <div class="page-pretitle">
                    Overview
                </div>
                <h2 class="page-title">
                    @yield('title',"CodeFec")
                </h2>
            </div>
            <!-- Page title actions -->
            <div class="col-auto ms-auto d-print-none">
                <div class="btn-list">
                    @foreach (\App\CodeFec\Header\functions::headerBtn() as $key => $value)
                        @include($value['view'])
                    @endforeach
                    @yield('headerBtn')
                </div>
            </div>
        </div>
    </div>
</div>
