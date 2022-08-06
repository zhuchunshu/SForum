@extends("App::app")

@section('title', __("topic.create"))

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            {{__("topic.create")}}
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection
@section('content')

    <div id="create-topic-vue">
        <form action="" method="post" @@submit.prevent="submit">
            <div class="row row-cards">
                <div class="col-md-12">
                    <div class="mb-3 border-0 card card-body">
                        <h3 class="card-title">{{__("app.title")}}</h3>
                        <input type="text" v-model="title" class="form-control form-control-lg form-control-flush"
                            placeholder="{{__("topic.Please enter a title")}}" required>
                        <h3 class="card-title">{{__("app.tag")}}</h3>
                        <div class="mb-3">
                            <select id="select-tags" v-model="tag_selected"
                                class="form-select form-select-lg form-control-flush">
                                <option v-for="option in tags" :data-custom-properties="option.icons" :value="option.value">
                                    @{{ option . text }}
                                </option>
                            </select>
                        </div>

                        <div class="mb-3">
                            <div class="row">
                                @if(count((new \App\Plugins\Core\src\Lib\Emoji())->get()))
                                    <div class="col-md-3">
                                        <div class="card">
                                            <ul class="nav nav-tabs" data-bs-toggle="tabs">
                                                @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                    <li class="nav-item">
                                                        <a href="#emoji-list-{{$key}}" class="nav-link @if ($loop->first) active @endif" data-bs-toggle="tab">{{$key}}</a>
                                                    </li>
                                                @endforeach
                                            </ul>
                                            <div class="card-body">
                                                <div class="tab-content">
                                                    @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                        <div class="tab-pane overflow-auto  @if ($loop->first) active @endif show" id="emoji-list-{{$key}}" style="max-height: 320px">
                                                            <div class="row">
                                                                @if($value['type'] === 'image')
                                                                    @foreach($value['container'] as $emojis)
                                                                        <div @@click="selectEmoji('{{$emojis['text']}}')" class="col-2 hvr-glow emoji-picker" emoji-data="{{$emojis['text']}}">{!! $emojis['icon'] !!}</div>
                                                                    @endforeach
                                                                @endif
                                                            </div>
                                                        </div>
                                                    @endforeach
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    <div class="col-md-9">
                                        <div id="content-vditor"></div>
                                    </div>
                                @else
                                    <div class="col-md-12">
                                        <div id="content-vditor"></div>
                                    </div>
                                @endif

                            </div>
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary">{{__("topic.publish")}}</button> Or <button type="button" @@click="draft" class="btn btn-danger">
                            {{__("topic.draft")}}</button>
                        </div>
                    </div>
                </div>

            </div>
        </form>
    </div>
    <div class="offcanvas offcanvas-start" data-bs-scroll="true" tabindex="-1" id="myOffcanvas" aria-labelledby="offcanvasWithBothOptionsLabel">
        <div class="offcanvas-header">
            <h5 class="offcanvas-title" id="offcanvasWithBothOptionsLabel">Backdrop with scrolling</h5>
            <button type="button" class="btn-close" data-bs-dismiss="offcanvas" aria-label="Close"></button>
        </div>
        <div class="offcanvas-body">
            <p>Try scrolling the rest of the page to see this option in action.</p>
        </div>
    </div>

@endsection

@section('scripts')
    <script type="text/javascript">
        var imageUpUrl = "/user/upload/image?_token={{ csrf_token() }}";
    </script>

    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection