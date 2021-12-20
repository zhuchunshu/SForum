<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateTopicVoteTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('topic_vote', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->text('title');
            $table->longText('options');
            $table->longText('data');
            $table->string('user_id');
            $table->string('_id');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('topic_vote');
    }
}
