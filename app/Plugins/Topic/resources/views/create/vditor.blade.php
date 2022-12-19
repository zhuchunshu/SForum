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
                            <option v-for="option in tags" :data-custom-properties="option.icons"
                                    :value="option.value">
                                @{{ option . text }}
                            </option>
                        </select>
                    </div>

                    <div class="mb-3">
                        <div class="row">
                            @if(get_options("topic_emoji_close",'false')!=="true" && count((new \App\Plugins\Core\src\Lib\Emoji())->get()))
                                <div class="col-md-3">
                                    <div class="card">
                                        <ul class="nav nav-tabs" data-bs-toggle="tabs" style="flex-wrap: inherit;
        width: 100%;
        height: 3.333333rem;
        padding: 0.373333rem 0.32rem 0;
        box-sizing: border-box;
        /* 下面是实现横向滚动的关键代码 */
        display: inline;
        float: left;
        white-space: nowrap;
        overflow-x: scroll;
        -webkit-overflow-scrolling: touch; /*解决在ios滑动不顺畅问题*/
        overflow-y: hidden;">
                                            @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                <li class="nav-item">
                                                    <a href="#emoji-list-{{$key}}"
                                                       class="nav-link @if ($loop->first) active @endif"
                                                       data-bs-toggle="tab">{{$key}}</a>
                                                </li>
                                            @endforeach
                                        </ul>
                                        <div class="card-body">
                                            <div class="tab-content">
                                                @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                    <div class="tab-pane  @if ($loop->first) active @endif show"
                                                         id="emoji-list-{{$key}}"
                                                         style="max-height: 320px;overflow-x: hidden;">
                                                        <div class="row">
                                                            @if($value['type'] === 'image')
                                                                @foreach($value['container'] as $emojis)
                                                                    <div @@click="selectEmoji('{{$emojis['text']}}')"
                                                                         class="col-3 col-sm-2 col-md-4 col-lg-3 hvr-glow emoji-picker"
                                                                         emoji-data="{{$emojis['text']}}">{!! $emojis['icon'] !!}</div>
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
                        <button class="btn btn-primary">{{__("topic.publish")}}</button>
                        Or
                        <button type="button" @@click="draft" class="btn btn-danger">
                            {{__("topic.draft")}}</button>
                    </div>
                </div>
            </div>

        </div>
    </form>
</div>
<div class="offcanvas offcanvas-start" data-bs-scroll="true" tabindex="-1" id="myOffcanvas"
     aria-labelledby="offcanvasWithBothOptionsLabel">
    <div class="offcanvas-header">
        <h5 class="offcanvas-title" id="offcanvasWithBothOptionsLabel">Backdrop with scrolling</h5>
        <button type="button" class="btn-close" data-bs-dismiss="offcanvas" aria-label="Close"></button>
    </div>
    <div class="offcanvas-body">
        <p>Try scrolling the rest of the page to see this option in action.</p>
    </div>
</div>