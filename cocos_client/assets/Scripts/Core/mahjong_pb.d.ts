export interface WSMessage {
  action?: string;
  data?: Uint8Array;
}

export function encodeWSMessage(message: WSMessage): Uint8Array {
  let bb = popByteBuffer();
  _encodeWSMessage(message, bb);
  return toUint8Array(bb);
}

function _encodeWSMessage(message: WSMessage, bb: ByteBuffer): void {
  // optional string action = 1;
  let $action = message.action;
  if ($action !== undefined) {
    writeVarint32(bb, 10);
    writeString(bb, $action);
  }

  // optional bytes data = 2;
  let $data = message.data;
  if ($data !== undefined) {
    writeVarint32(bb, 18);
    writeVarint32(bb, $data.length), writeBytes(bb, $data);
  }
}

export function decodeWSMessage(binary: Uint8Array): WSMessage {
  return _decodeWSMessage(wrapByteBuffer(binary));
}

function _decodeWSMessage(bb: ByteBuffer): WSMessage {
  let message: WSMessage = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional string action = 1;
      case 1: {
        message.action = readString(bb, readVarint32(bb));
        break;
      }

      // optional bytes data = 2;
      case 2: {
        message.data = readBytes(bb, readVarint32(bb));
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface PlayerInfo {
  id?: string;
  name?: string;
  seat?: number;
  score?: number;
  hand_tiles?: number[];
  discarded_tiles?: number[];
  melds?: MeldData[];
}

export function encodePlayerInfo(message: PlayerInfo): Uint8Array {
  let bb = popByteBuffer();
  _encodePlayerInfo(message, bb);
  return toUint8Array(bb);
}

function _encodePlayerInfo(message: PlayerInfo, bb: ByteBuffer): void {
  // optional string id = 1;
  let $id = message.id;
  if ($id !== undefined) {
    writeVarint32(bb, 10);
    writeString(bb, $id);
  }

  // optional string name = 2;
  let $name = message.name;
  if ($name !== undefined) {
    writeVarint32(bb, 18);
    writeString(bb, $name);
  }

  // optional int32 seat = 3;
  let $seat = message.seat;
  if ($seat !== undefined) {
    writeVarint32(bb, 24);
    writeVarint64(bb, intToLong($seat));
  }

  // optional int32 score = 4;
  let $score = message.score;
  if ($score !== undefined) {
    writeVarint32(bb, 32);
    writeVarint64(bb, intToLong($score));
  }

  // repeated int32 hand_tiles = 5;
  let array$hand_tiles = message.hand_tiles;
  if (array$hand_tiles !== undefined) {
    let packed = popByteBuffer();
    for (let value of array$hand_tiles) {
      writeVarint64(packed, intToLong(value));
    }
    writeVarint32(bb, 42);
    writeVarint32(bb, packed.offset);
    writeByteBuffer(bb, packed);
    pushByteBuffer(packed);
  }

  // repeated int32 discarded_tiles = 6;
  let array$discarded_tiles = message.discarded_tiles;
  if (array$discarded_tiles !== undefined) {
    let packed = popByteBuffer();
    for (let value of array$discarded_tiles) {
      writeVarint64(packed, intToLong(value));
    }
    writeVarint32(bb, 50);
    writeVarint32(bb, packed.offset);
    writeByteBuffer(bb, packed);
    pushByteBuffer(packed);
  }

  // repeated MeldData melds = 7;
  let array$melds = message.melds;
  if (array$melds !== undefined) {
    for (let value of array$melds) {
      writeVarint32(bb, 58);
      let nested = popByteBuffer();
      _encodeMeldData(value, nested);
      writeVarint32(bb, nested.limit);
      writeByteBuffer(bb, nested);
      pushByteBuffer(nested);
    }
  }
}

export function decodePlayerInfo(binary: Uint8Array): PlayerInfo {
  return _decodePlayerInfo(wrapByteBuffer(binary));
}

function _decodePlayerInfo(bb: ByteBuffer): PlayerInfo {
  let message: PlayerInfo = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional string id = 1;
      case 1: {
        message.id = readString(bb, readVarint32(bb));
        break;
      }

      // optional string name = 2;
      case 2: {
        message.name = readString(bb, readVarint32(bb));
        break;
      }

      // optional int32 seat = 3;
      case 3: {
        message.seat = readVarint32(bb);
        break;
      }

      // optional int32 score = 4;
      case 4: {
        message.score = readVarint32(bb);
        break;
      }

      // repeated int32 hand_tiles = 5;
      case 5: {
        let values = message.hand_tiles || (message.hand_tiles = []);
        if ((tag & 7) === 2) {
          let outerLimit = pushTemporaryLength(bb);
          while (!isAtEnd(bb)) {
            values.push(readVarint32(bb));
          }
          bb.limit = outerLimit;
        } else {
          values.push(readVarint32(bb));
        }
        break;
      }

      // repeated int32 discarded_tiles = 6;
      case 6: {
        let values = message.discarded_tiles || (message.discarded_tiles = []);
        if ((tag & 7) === 2) {
          let outerLimit = pushTemporaryLength(bb);
          while (!isAtEnd(bb)) {
            values.push(readVarint32(bb));
          }
          bb.limit = outerLimit;
        } else {
          values.push(readVarint32(bb));
        }
        break;
      }

      // repeated MeldData melds = 7;
      case 7: {
        let limit = pushTemporaryLength(bb);
        let values = message.melds || (message.melds = []);
        values.push(_decodeMeldData(bb));
        bb.limit = limit;
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface MeldData {
  type?: number;
  tiles?: number[];
}

export function encodeMeldData(message: MeldData): Uint8Array {
  let bb = popByteBuffer();
  _encodeMeldData(message, bb);
  return toUint8Array(bb);
}

function _encodeMeldData(message: MeldData, bb: ByteBuffer): void {
  // optional int32 type = 1;
  let $type = message.type;
  if ($type !== undefined) {
    writeVarint32(bb, 8);
    writeVarint64(bb, intToLong($type));
  }

  // repeated int32 tiles = 2;
  let array$tiles = message.tiles;
  if (array$tiles !== undefined) {
    let packed = popByteBuffer();
    for (let value of array$tiles) {
      writeVarint64(packed, intToLong(value));
    }
    writeVarint32(bb, 18);
    writeVarint32(bb, packed.offset);
    writeByteBuffer(bb, packed);
    pushByteBuffer(packed);
  }
}

export function decodeMeldData(binary: Uint8Array): MeldData {
  return _decodeMeldData(wrapByteBuffer(binary));
}

function _decodeMeldData(bb: ByteBuffer): MeldData {
  let message: MeldData = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional int32 type = 1;
      case 1: {
        message.type = readVarint32(bb);
        break;
      }

      // repeated int32 tiles = 2;
      case 2: {
        let values = message.tiles || (message.tiles = []);
        if ((tag & 7) === 2) {
          let outerLimit = pushTemporaryLength(bb);
          while (!isAtEnd(bb)) {
            values.push(readVarint32(bb));
          }
          bb.limit = outerLimit;
        } else {
          values.push(readVarint32(bb));
        }
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface SyncStateData {
  room_id?: string;
  current_wind?: number;
  remaining_tiles?: number;
  current_turn_player_id?: string;
  game_state?: string;
  players?: PlayerInfo[];
}

export function encodeSyncStateData(message: SyncStateData): Uint8Array {
  let bb = popByteBuffer();
  _encodeSyncStateData(message, bb);
  return toUint8Array(bb);
}

function _encodeSyncStateData(message: SyncStateData, bb: ByteBuffer): void {
  // optional string room_id = 1;
  let $room_id = message.room_id;
  if ($room_id !== undefined) {
    writeVarint32(bb, 10);
    writeString(bb, $room_id);
  }

  // optional int32 current_wind = 2;
  let $current_wind = message.current_wind;
  if ($current_wind !== undefined) {
    writeVarint32(bb, 16);
    writeVarint64(bb, intToLong($current_wind));
  }

  // optional int32 remaining_tiles = 3;
  let $remaining_tiles = message.remaining_tiles;
  if ($remaining_tiles !== undefined) {
    writeVarint32(bb, 24);
    writeVarint64(bb, intToLong($remaining_tiles));
  }

  // optional string current_turn_player_id = 4;
  let $current_turn_player_id = message.current_turn_player_id;
  if ($current_turn_player_id !== undefined) {
    writeVarint32(bb, 34);
    writeString(bb, $current_turn_player_id);
  }

  // optional string game_state = 5;
  let $game_state = message.game_state;
  if ($game_state !== undefined) {
    writeVarint32(bb, 42);
    writeString(bb, $game_state);
  }

  // repeated PlayerInfo players = 6;
  let array$players = message.players;
  if (array$players !== undefined) {
    for (let value of array$players) {
      writeVarint32(bb, 50);
      let nested = popByteBuffer();
      _encodePlayerInfo(value, nested);
      writeVarint32(bb, nested.limit);
      writeByteBuffer(bb, nested);
      pushByteBuffer(nested);
    }
  }
}

export function decodeSyncStateData(binary: Uint8Array): SyncStateData {
  return _decodeSyncStateData(wrapByteBuffer(binary));
}

function _decodeSyncStateData(bb: ByteBuffer): SyncStateData {
  let message: SyncStateData = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional string room_id = 1;
      case 1: {
        message.room_id = readString(bb, readVarint32(bb));
        break;
      }

      // optional int32 current_wind = 2;
      case 2: {
        message.current_wind = readVarint32(bb);
        break;
      }

      // optional int32 remaining_tiles = 3;
      case 3: {
        message.remaining_tiles = readVarint32(bb);
        break;
      }

      // optional string current_turn_player_id = 4;
      case 4: {
        message.current_turn_player_id = readString(bb, readVarint32(bb));
        break;
      }

      // optional string game_state = 5;
      case 5: {
        message.game_state = readString(bb, readVarint32(bb));
        break;
      }

      // repeated PlayerInfo players = 6;
      case 6: {
        let limit = pushTemporaryLength(bb);
        let values = message.players || (message.players = []);
        values.push(_decodePlayerInfo(bb));
        bb.limit = limit;
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface DealTilesData {
  tiles?: number[];
}

export function encodeDealTilesData(message: DealTilesData): Uint8Array {
  let bb = popByteBuffer();
  _encodeDealTilesData(message, bb);
  return toUint8Array(bb);
}

function _encodeDealTilesData(message: DealTilesData, bb: ByteBuffer): void {
  // repeated int32 tiles = 1;
  let array$tiles = message.tiles;
  if (array$tiles !== undefined) {
    let packed = popByteBuffer();
    for (let value of array$tiles) {
      writeVarint64(packed, intToLong(value));
    }
    writeVarint32(bb, 10);
    writeVarint32(bb, packed.offset);
    writeByteBuffer(bb, packed);
    pushByteBuffer(packed);
  }
}

export function decodeDealTilesData(binary: Uint8Array): DealTilesData {
  return _decodeDealTilesData(wrapByteBuffer(binary));
}

function _decodeDealTilesData(bb: ByteBuffer): DealTilesData {
  let message: DealTilesData = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // repeated int32 tiles = 1;
      case 1: {
        let values = message.tiles || (message.tiles = []);
        if ((tag & 7) === 2) {
          let outerLimit = pushTemporaryLength(bb);
          while (!isAtEnd(bb)) {
            values.push(readVarint32(bb));
          }
          bb.limit = outerLimit;
        } else {
          values.push(readVarint32(bb));
        }
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface ActionBroadcastData {
  player_id?: string;
  action_type?: number;
  tile_id?: number;
  related_tiles?: number[];
}

export function encodeActionBroadcastData(message: ActionBroadcastData): Uint8Array {
  let bb = popByteBuffer();
  _encodeActionBroadcastData(message, bb);
  return toUint8Array(bb);
}

function _encodeActionBroadcastData(message: ActionBroadcastData, bb: ByteBuffer): void {
  // optional string player_id = 1;
  let $player_id = message.player_id;
  if ($player_id !== undefined) {
    writeVarint32(bb, 10);
    writeString(bb, $player_id);
  }

  // optional int32 action_type = 2;
  let $action_type = message.action_type;
  if ($action_type !== undefined) {
    writeVarint32(bb, 16);
    writeVarint64(bb, intToLong($action_type));
  }

  // optional int32 tile_id = 3;
  let $tile_id = message.tile_id;
  if ($tile_id !== undefined) {
    writeVarint32(bb, 24);
    writeVarint64(bb, intToLong($tile_id));
  }

  // repeated int32 related_tiles = 4;
  let array$related_tiles = message.related_tiles;
  if (array$related_tiles !== undefined) {
    let packed = popByteBuffer();
    for (let value of array$related_tiles) {
      writeVarint64(packed, intToLong(value));
    }
    writeVarint32(bb, 34);
    writeVarint32(bb, packed.offset);
    writeByteBuffer(bb, packed);
    pushByteBuffer(packed);
  }
}

export function decodeActionBroadcastData(binary: Uint8Array): ActionBroadcastData {
  return _decodeActionBroadcastData(wrapByteBuffer(binary));
}

function _decodeActionBroadcastData(bb: ByteBuffer): ActionBroadcastData {
  let message: ActionBroadcastData = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional string player_id = 1;
      case 1: {
        message.player_id = readString(bb, readVarint32(bb));
        break;
      }

      // optional int32 action_type = 2;
      case 2: {
        message.action_type = readVarint32(bb);
        break;
      }

      // optional int32 tile_id = 3;
      case 3: {
        message.tile_id = readVarint32(bb);
        break;
      }

      // repeated int32 related_tiles = 4;
      case 4: {
        let values = message.related_tiles || (message.related_tiles = []);
        if ((tag & 7) === 2) {
          let outerLimit = pushTemporaryLength(bb);
          while (!isAtEnd(bb)) {
            values.push(readVarint32(bb));
          }
          bb.limit = outerLimit;
        } else {
          values.push(readVarint32(bb));
        }
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface JoinRoomReq {
  room_id?: string;
  player_id?: string;
}

export function encodeJoinRoomReq(message: JoinRoomReq): Uint8Array {
  let bb = popByteBuffer();
  _encodeJoinRoomReq(message, bb);
  return toUint8Array(bb);
}

function _encodeJoinRoomReq(message: JoinRoomReq, bb: ByteBuffer): void {
  // optional string room_id = 1;
  let $room_id = message.room_id;
  if ($room_id !== undefined) {
    writeVarint32(bb, 10);
    writeString(bb, $room_id);
  }

  // optional string player_id = 2;
  let $player_id = message.player_id;
  if ($player_id !== undefined) {
    writeVarint32(bb, 18);
    writeString(bb, $player_id);
  }
}

export function decodeJoinRoomReq(binary: Uint8Array): JoinRoomReq {
  return _decodeJoinRoomReq(wrapByteBuffer(binary));
}

function _decodeJoinRoomReq(bb: ByteBuffer): JoinRoomReq {
  let message: JoinRoomReq = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional string room_id = 1;
      case 1: {
        message.room_id = readString(bb, readVarint32(bb));
        break;
      }

      // optional string player_id = 2;
      case 2: {
        message.player_id = readString(bb, readVarint32(bb));
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface JoinRoomRes {
  success?: boolean;
  message?: string;
}

export function encodeJoinRoomRes(message: JoinRoomRes): Uint8Array {
  let bb = popByteBuffer();
  _encodeJoinRoomRes(message, bb);
  return toUint8Array(bb);
}

function _encodeJoinRoomRes(message: JoinRoomRes, bb: ByteBuffer): void {
  // optional bool success = 1;
  let $success = message.success;
  if ($success !== undefined) {
    writeVarint32(bb, 8);
    writeByte(bb, $success ? 1 : 0);
  }

  // optional string message = 2;
  let $message = message.message;
  if ($message !== undefined) {
    writeVarint32(bb, 18);
    writeString(bb, $message);
  }
}

export function decodeJoinRoomRes(binary: Uint8Array): JoinRoomRes {
  return _decodeJoinRoomRes(wrapByteBuffer(binary));
}

function _decodeJoinRoomRes(bb: ByteBuffer): JoinRoomRes {
  let message: JoinRoomRes = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional bool success = 1;
      case 1: {
        message.success = !!readByte(bb);
        break;
      }

      // optional string message = 2;
      case 2: {
        message.message = readString(bb, readVarint32(bb));
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface PlayerActionData {
  action_type?: number;
  tile_id?: number;
}

export function encodePlayerActionData(message: PlayerActionData): Uint8Array {
  let bb = popByteBuffer();
  _encodePlayerActionData(message, bb);
  return toUint8Array(bb);
}

function _encodePlayerActionData(message: PlayerActionData, bb: ByteBuffer): void {
  // optional int32 action_type = 1;
  let $action_type = message.action_type;
  if ($action_type !== undefined) {
    writeVarint32(bb, 8);
    writeVarint64(bb, intToLong($action_type));
  }

  // optional int32 tile_id = 2;
  let $tile_id = message.tile_id;
  if ($tile_id !== undefined) {
    writeVarint32(bb, 16);
    writeVarint64(bb, intToLong($tile_id));
  }
}

export function decodePlayerActionData(binary: Uint8Array): PlayerActionData {
  return _decodePlayerActionData(wrapByteBuffer(binary));
}

function _decodePlayerActionData(bb: ByteBuffer): PlayerActionData {
  let message: PlayerActionData = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional int32 action_type = 1;
      case 1: {
        message.action_type = readVarint32(bb);
        break;
      }

      // optional int32 tile_id = 2;
      case 2: {
        message.tile_id = readVarint32(bb);
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface PlayerActionRes {
  success?: boolean;
  message?: string;
}

export function encodePlayerActionRes(message: PlayerActionRes): Uint8Array {
  let bb = popByteBuffer();
  _encodePlayerActionRes(message, bb);
  return toUint8Array(bb);
}

function _encodePlayerActionRes(message: PlayerActionRes, bb: ByteBuffer): void {
  // optional bool success = 1;
  let $success = message.success;
  if ($success !== undefined) {
    writeVarint32(bb, 8);
    writeByte(bb, $success ? 1 : 0);
  }

  // optional string message = 2;
  let $message = message.message;
  if ($message !== undefined) {
    writeVarint32(bb, 18);
    writeString(bb, $message);
  }
}

export function decodePlayerActionRes(binary: Uint8Array): PlayerActionRes {
  return _decodePlayerActionRes(wrapByteBuffer(binary));
}

function _decodePlayerActionRes(bb: ByteBuffer): PlayerActionRes {
  let message: PlayerActionRes = {} as any;

  end_of_message: while (!isAtEnd(bb)) {
    let tag = readVarint32(bb);

    switch (tag >>> 3) {
      case 0:
        break end_of_message;

      // optional bool success = 1;
      case 1: {
        message.success = !!readByte(bb);
        break;
      }

      // optional string message = 2;
      case 2: {
        message.message = readString(bb, readVarint32(bb));
        break;
      }

      default:
        skipUnknownField(bb, tag & 7);
    }
  }

  return message;
}

export interface Long {
  low: number;
  high: number;
  unsigned: boolean;
}

interface ByteBuffer {
  bytes: Uint8Array;
  offset: number;
  limit: number;
}

function pushTemporaryLength(bb: ByteBuffer): number {
  let length = readVarint32(bb);
  let limit = bb.limit;
  bb.limit = bb.offset + length;
  return limit;
}

function skipUnknownField(bb: ByteBuffer, type: number): void {
  switch (type) {
    case 0: while (readByte(bb) & 0x80) { } break;
    case 2: skip(bb, readVarint32(bb)); break;
    case 5: skip(bb, 4); break;
    case 1: skip(bb, 8); break;
    default: throw new Error("Unimplemented type: " + type);
  }
}

function stringToLong(value: string): Long {
  return {
    low: value.charCodeAt(0) | (value.charCodeAt(1) << 16),
    high: value.charCodeAt(2) | (value.charCodeAt(3) << 16),
    unsigned: false,
  };
}

function longToString(value: Long): string {
  let low = value.low;
  let high = value.high;
  return String.fromCharCode(
    low & 0xFFFF,
    low >>> 16,
    high & 0xFFFF,
    high >>> 16);
}

// The code below was modified from https://github.com/protobufjs/bytebuffer.js
// which is under the Apache License 2.0.

let f32 = new Float32Array(1);
let f32_u8 = new Uint8Array(f32.buffer);

let f64 = new Float64Array(1);
let f64_u8 = new Uint8Array(f64.buffer);

function intToLong(value: number): Long {
  value |= 0;
  return {
    low: value,
    high: value >> 31,
    unsigned: value >= 0,
  };
}

let bbStack: ByteBuffer[] = [];

function popByteBuffer(): ByteBuffer {
  const bb = bbStack.pop();
  if (!bb) return { bytes: new Uint8Array(64), offset: 0, limit: 0 };
  bb.offset = bb.limit = 0;
  return bb;
}

function pushByteBuffer(bb: ByteBuffer): void {
  bbStack.push(bb);
}

function wrapByteBuffer(bytes: Uint8Array): ByteBuffer {
  return { bytes, offset: 0, limit: bytes.length };
}

function toUint8Array(bb: ByteBuffer): Uint8Array {
  let bytes = bb.bytes;
  let limit = bb.limit;
  return bytes.length === limit ? bytes : bytes.subarray(0, limit);
}

function skip(bb: ByteBuffer, offset: number): void {
  if (bb.offset + offset > bb.limit) {
    throw new Error('Skip past limit');
  }
  bb.offset += offset;
}

function isAtEnd(bb: ByteBuffer): boolean {
  return bb.offset >= bb.limit;
}

function grow(bb: ByteBuffer, count: number): number {
  let bytes = bb.bytes;
  let offset = bb.offset;
  let limit = bb.limit;
  let finalOffset = offset + count;
  if (finalOffset > bytes.length) {
    let newBytes = new Uint8Array(finalOffset * 2);
    newBytes.set(bytes);
    bb.bytes = newBytes;
  }
  bb.offset = finalOffset;
  if (finalOffset > limit) {
    bb.limit = finalOffset;
  }
  return offset;
}

function advance(bb: ByteBuffer, count: number): number {
  let offset = bb.offset;
  if (offset + count > bb.limit) {
    throw new Error('Read past limit');
  }
  bb.offset += count;
  return offset;
}

function readBytes(bb: ByteBuffer, count: number): Uint8Array {
  let offset = advance(bb, count);
  return bb.bytes.subarray(offset, offset + count);
}

function writeBytes(bb: ByteBuffer, buffer: Uint8Array): void {
  let offset = grow(bb, buffer.length);
  bb.bytes.set(buffer, offset);
}

function readString(bb: ByteBuffer, count: number): string {
  // Sadly a hand-coded UTF8 decoder is much faster than subarray+TextDecoder in V8
  let offset = advance(bb, count);
  let fromCharCode = String.fromCharCode;
  let bytes = bb.bytes;
  let invalid = '\uFFFD';
  let text = '';

  for (let i = 0; i < count; i++) {
    let c1 = bytes[i + offset], c2: number, c3: number, c4: number, c: number;

    // 1 byte
    if ((c1 & 0x80) === 0) {
      text += fromCharCode(c1);
    }

    // 2 bytes
    else if ((c1 & 0xE0) === 0xC0) {
      if (i + 1 >= count) text += invalid;
      else {
        c2 = bytes[i + offset + 1];
        if ((c2 & 0xC0) !== 0x80) text += invalid;
        else {
          c = ((c1 & 0x1F) << 6) | (c2 & 0x3F);
          if (c < 0x80) text += invalid;
          else {
            text += fromCharCode(c);
            i++;
          }
        }
      }
    }

    // 3 bytes
    else if ((c1 & 0xF0) == 0xE0) {
      if (i + 2 >= count) text += invalid;
      else {
        c2 = bytes[i + offset + 1];
        c3 = bytes[i + offset + 2];
        if (((c2 | (c3 << 8)) & 0xC0C0) !== 0x8080) text += invalid;
        else {
          c = ((c1 & 0x0F) << 12) | ((c2 & 0x3F) << 6) | (c3 & 0x3F);
          if (c < 0x0800 || (c >= 0xD800 && c <= 0xDFFF)) text += invalid;
          else {
            text += fromCharCode(c);
            i += 2;
          }
        }
      }
    }

    // 4 bytes
    else if ((c1 & 0xF8) == 0xF0) {
      if (i + 3 >= count) text += invalid;
      else {
        c2 = bytes[i + offset + 1];
        c3 = bytes[i + offset + 2];
        c4 = bytes[i + offset + 3];
        if (((c2 | (c3 << 8) | (c4 << 16)) & 0xC0C0C0) !== 0x808080) text += invalid;
        else {
          c = ((c1 & 0x07) << 0x12) | ((c2 & 0x3F) << 0x0C) | ((c3 & 0x3F) << 0x06) | (c4 & 0x3F);
          if (c < 0x10000 || c > 0x10FFFF) text += invalid;
          else {
            c -= 0x10000;
            text += fromCharCode((c >> 10) + 0xD800, (c & 0x3FF) + 0xDC00);
            i += 3;
          }
        }
      }
    }

    else text += invalid;
  }

  return text;
}

function writeString(bb: ByteBuffer, text: string): void {
  // Sadly a hand-coded UTF8 encoder is much faster than TextEncoder+set in V8
  let n = text.length;
  let byteCount = 0;

  // Write the byte count first
  for (let i = 0; i < n; i++) {
    let c = text.charCodeAt(i);
    if (c >= 0xD800 && c <= 0xDBFF && i + 1 < n) {
      c = (c << 10) + text.charCodeAt(++i) - 0x35FDC00;
    }
    byteCount += c < 0x80 ? 1 : c < 0x800 ? 2 : c < 0x10000 ? 3 : 4;
  }
  writeVarint32(bb, byteCount);

  let offset = grow(bb, byteCount);
  let bytes = bb.bytes;

  // Then write the bytes
  for (let i = 0; i < n; i++) {
    let c = text.charCodeAt(i);
    if (c >= 0xD800 && c <= 0xDBFF && i + 1 < n) {
      c = (c << 10) + text.charCodeAt(++i) - 0x35FDC00;
    }
    if (c < 0x80) {
      bytes[offset++] = c;
    } else {
      if (c < 0x800) {
        bytes[offset++] = ((c >> 6) & 0x1F) | 0xC0;
      } else {
        if (c < 0x10000) {
          bytes[offset++] = ((c >> 12) & 0x0F) | 0xE0;
        } else {
          bytes[offset++] = ((c >> 18) & 0x07) | 0xF0;
          bytes[offset++] = ((c >> 12) & 0x3F) | 0x80;
        }
        bytes[offset++] = ((c >> 6) & 0x3F) | 0x80;
      }
      bytes[offset++] = (c & 0x3F) | 0x80;
    }
  }
}

function writeByteBuffer(bb: ByteBuffer, buffer: ByteBuffer): void {
  let offset = grow(bb, buffer.limit);
  let from = bb.bytes;
  let to = buffer.bytes;

  // This for loop is much faster than subarray+set on V8
  for (let i = 0, n = buffer.limit; i < n; i++) {
    from[i + offset] = to[i];
  }
}

function readByte(bb: ByteBuffer): number {
  return bb.bytes[advance(bb, 1)];
}

function writeByte(bb: ByteBuffer, value: number): void {
  let offset = grow(bb, 1);
  bb.bytes[offset] = value;
}

function readFloat(bb: ByteBuffer): number {
  let offset = advance(bb, 4);
  let bytes = bb.bytes;

  // Manual copying is much faster than subarray+set in V8
  f32_u8[0] = bytes[offset++];
  f32_u8[1] = bytes[offset++];
  f32_u8[2] = bytes[offset++];
  f32_u8[3] = bytes[offset++];
  return f32[0];
}

function writeFloat(bb: ByteBuffer, value: number): void {
  let offset = grow(bb, 4);
  let bytes = bb.bytes;
  f32[0] = value;

  // Manual copying is much faster than subarray+set in V8
  bytes[offset++] = f32_u8[0];
  bytes[offset++] = f32_u8[1];
  bytes[offset++] = f32_u8[2];
  bytes[offset++] = f32_u8[3];
}

function readDouble(bb: ByteBuffer): number {
  let offset = advance(bb, 8);
  let bytes = bb.bytes;

  // Manual copying is much faster than subarray+set in V8
  f64_u8[0] = bytes[offset++];
  f64_u8[1] = bytes[offset++];
  f64_u8[2] = bytes[offset++];
  f64_u8[3] = bytes[offset++];
  f64_u8[4] = bytes[offset++];
  f64_u8[5] = bytes[offset++];
  f64_u8[6] = bytes[offset++];
  f64_u8[7] = bytes[offset++];
  return f64[0];
}

function writeDouble(bb: ByteBuffer, value: number): void {
  let offset = grow(bb, 8);
  let bytes = bb.bytes;
  f64[0] = value;

  // Manual copying is much faster than subarray+set in V8
  bytes[offset++] = f64_u8[0];
  bytes[offset++] = f64_u8[1];
  bytes[offset++] = f64_u8[2];
  bytes[offset++] = f64_u8[3];
  bytes[offset++] = f64_u8[4];
  bytes[offset++] = f64_u8[5];
  bytes[offset++] = f64_u8[6];
  bytes[offset++] = f64_u8[7];
}

function readInt32(bb: ByteBuffer): number {
  let offset = advance(bb, 4);
  let bytes = bb.bytes;
  return (
    bytes[offset] |
    (bytes[offset + 1] << 8) |
    (bytes[offset + 2] << 16) |
    (bytes[offset + 3] << 24)
  );
}

function writeInt32(bb: ByteBuffer, value: number): void {
  let offset = grow(bb, 4);
  let bytes = bb.bytes;
  bytes[offset] = value;
  bytes[offset + 1] = value >> 8;
  bytes[offset + 2] = value >> 16;
  bytes[offset + 3] = value >> 24;
}

function readInt64(bb: ByteBuffer, unsigned: boolean): Long {
  return {
    low: readInt32(bb),
    high: readInt32(bb),
    unsigned,
  };
}

function writeInt64(bb: ByteBuffer, value: Long): void {
  writeInt32(bb, value.low);
  writeInt32(bb, value.high);
}

function readVarint32(bb: ByteBuffer): number {
  let c = 0;
  let value = 0;
  let b: number;
  do {
    b = readByte(bb);
    if (c < 32) value |= (b & 0x7F) << c;
    c += 7;
  } while (b & 0x80);
  return value;
}

function writeVarint32(bb: ByteBuffer, value: number): void {
  value >>>= 0;
  while (value >= 0x80) {
    writeByte(bb, (value & 0x7f) | 0x80);
    value >>>= 7;
  }
  writeByte(bb, value);
}

function readVarint64(bb: ByteBuffer, unsigned: boolean): Long {
  let part0 = 0;
  let part1 = 0;
  let part2 = 0;
  let b: number;

  b = readByte(bb); part0 = (b & 0x7F); if (b & 0x80) {
    b = readByte(bb); part0 |= (b & 0x7F) << 7; if (b & 0x80) {
      b = readByte(bb); part0 |= (b & 0x7F) << 14; if (b & 0x80) {
        b = readByte(bb); part0 |= (b & 0x7F) << 21; if (b & 0x80) {

          b = readByte(bb); part1 = (b & 0x7F); if (b & 0x80) {
            b = readByte(bb); part1 |= (b & 0x7F) << 7; if (b & 0x80) {
              b = readByte(bb); part1 |= (b & 0x7F) << 14; if (b & 0x80) {
                b = readByte(bb); part1 |= (b & 0x7F) << 21; if (b & 0x80) {

                  b = readByte(bb); part2 = (b & 0x7F); if (b & 0x80) {
                    b = readByte(bb); part2 |= (b & 0x7F) << 7;
                  }
                }
              }
            }
          }
        }
      }
    }
  }

  return {
    low: part0 | (part1 << 28),
    high: (part1 >>> 4) | (part2 << 24),
    unsigned,
  };
}

function writeVarint64(bb: ByteBuffer, value: Long): void {
  let part0 = value.low >>> 0;
  let part1 = ((value.low >>> 28) | (value.high << 4)) >>> 0;
  let part2 = value.high >>> 24;

  // ref: src/google/protobuf/io/coded_stream.cc
  let size =
    part2 === 0 ?
      part1 === 0 ?
        part0 < 1 << 14 ?
          part0 < 1 << 7 ? 1 : 2 :
          part0 < 1 << 21 ? 3 : 4 :
        part1 < 1 << 14 ?
          part1 < 1 << 7 ? 5 : 6 :
          part1 < 1 << 21 ? 7 : 8 :
      part2 < 1 << 7 ? 9 : 10;

  let offset = grow(bb, size);
  let bytes = bb.bytes;

  switch (size) {
    case 10: bytes[offset + 9] = (part2 >>> 7) & 0x01;
    case 9: bytes[offset + 8] = size !== 9 ? part2 | 0x80 : part2 & 0x7F;
    case 8: bytes[offset + 7] = size !== 8 ? (part1 >>> 21) | 0x80 : (part1 >>> 21) & 0x7F;
    case 7: bytes[offset + 6] = size !== 7 ? (part1 >>> 14) | 0x80 : (part1 >>> 14) & 0x7F;
    case 6: bytes[offset + 5] = size !== 6 ? (part1 >>> 7) | 0x80 : (part1 >>> 7) & 0x7F;
    case 5: bytes[offset + 4] = size !== 5 ? part1 | 0x80 : part1 & 0x7F;
    case 4: bytes[offset + 3] = size !== 4 ? (part0 >>> 21) | 0x80 : (part0 >>> 21) & 0x7F;
    case 3: bytes[offset + 2] = size !== 3 ? (part0 >>> 14) | 0x80 : (part0 >>> 14) & 0x7F;
    case 2: bytes[offset + 1] = size !== 2 ? (part0 >>> 7) | 0x80 : (part0 >>> 7) & 0x7F;
    case 1: bytes[offset] = size !== 1 ? part0 | 0x80 : part0 & 0x7F;
  }
}

function readVarint32ZigZag(bb: ByteBuffer): number {
  let value = readVarint32(bb);

  // ref: src/google/protobuf/wire_format_lite.h
  return (value >>> 1) ^ -(value & 1);
}

function writeVarint32ZigZag(bb: ByteBuffer, value: number): void {
  // ref: src/google/protobuf/wire_format_lite.h
  writeVarint32(bb, (value << 1) ^ (value >> 31));
}

function readVarint64ZigZag(bb: ByteBuffer): Long {
  let value = readVarint64(bb, /* unsigned */ false);
  let low = value.low;
  let high = value.high;
  let flip = -(low & 1);

  // ref: src/google/protobuf/wire_format_lite.h
  return {
    low: ((low >>> 1) | (high << 31)) ^ flip,
    high: (high >>> 1) ^ flip,
    unsigned: false,
  };
}

function writeVarint64ZigZag(bb: ByteBuffer, value: Long): void {
  let low = value.low;
  let high = value.high;
  let flip = high >> 31;

  // ref: src/google/protobuf/wire_format_lite.h
  writeVarint64(bb, {
    low: (low << 1) ^ flip,
    high: ((high << 1) | (low >>> 31)) ^ flip,
    unsigned: false,
  });
}
declare namespace mahjong_pb {
  export function encodeWSMessage(message: WSMessage): Uint8Array;
  export function decodeWSMessage(binary: Uint8Array): WSMessage;
  export function encodePlayerInfo(message: PlayerInfo): Uint8Array;
  export function decodePlayerInfo(binary: Uint8Array): PlayerInfo;
  export function encodeMeldData(message: MeldData): Uint8Array;
  export function decodeMeldData(binary: Uint8Array): MeldData;
  export function encodeSyncStateData(message: SyncStateData): Uint8Array;
  export function decodeSyncStateData(binary: Uint8Array): SyncStateData;
  export function encodeDealTilesData(message: DealTilesData): Uint8Array;
  export function decodeDealTilesData(binary: Uint8Array): DealTilesData;
  export function encodeActionBroadcastData(message: ActionBroadcastData): Uint8Array;
  export function decodeActionBroadcastData(binary: Uint8Array): ActionBroadcastData;
  export function encodeJoinRoomReq(message: JoinRoomReq): Uint8Array;
  export function decodeJoinRoomReq(binary: Uint8Array): JoinRoomReq;
  export function encodeJoinRoomRes(message: JoinRoomRes): Uint8Array;
  export function decodeJoinRoomRes(binary: Uint8Array): JoinRoomRes;
  export function encodePlayerActionData(message: PlayerActionData): Uint8Array;
  export function decodePlayerActionData(binary: Uint8Array): PlayerActionData;
  export function encodePlayerActionRes(message: PlayerActionRes): Uint8Array;
  export function decodePlayerActionRes(binary: Uint8Array): PlayerActionRes;
}

declare global {
  interface Window {
    mahjong_pb: typeof mahjong_pb;
  }
}
export { };
